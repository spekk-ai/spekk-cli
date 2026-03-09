import { WebSocketServer } from 'ws';
import { spawn } from 'node:child_process';
import { createServer } from 'node:http';
import { launchAgentWithPrompt } from '../cli/prompt-resolver.js';
import { resolveSkillContent } from '../coach/cli.js';
import {
  formatChatMessage,
  formatElementSelection,
  formatActionRecording,
  formatInitMessage,
} from './message-formatter.js';
import { createWSAdapter } from './ws-adapter.js';
import { createServeApi } from './shapes.js';

const DEFAULT_PORT = 3118;

/**
 * Build the system prompt for the serve coach session.
 * Uses the real coach-agent prompt and appends coordinator skill as available context.
 */
function buildServeCoachPrompt() {
  const { activationMessage } = launchAgentWithPrompt('coach-agent');

  let prompt = activationMessage;

  const coordinatorSkill = resolveSkillContent('coordinate');
  if (coordinatorSkill) {
    prompt += '\n\n---\n\n**Available Skill: Coordinator**\n\n';
    prompt += 'The coordinator skill is available for use when the user asks to plan work, organize branches, or analyze dependencies.\n';
    prompt += 'Use it when you detect relevant triggers — do NOT execute it automatically on session start.\n';
    prompt += '\n<skill-reference>\n' + coordinatorSkill + '\n</skill-reference>\n';
  }

  return prompt;
}

/**
 * Map Claude tool names to user-friendly activity descriptions.
 */
function describeToolUse(toolName) {
  switch (toolName) {
    case 'Read':
    case 'Glob':
    case 'Grep':
      return 'Reading files...';
    case 'Write':
    case 'Edit':
    case 'NotebookEdit':
      return 'Writing code...';
    case 'Bash':
      return 'Running a command...';
    case 'WebSearch':
    case 'WebFetch':
      return 'Searching the web...';
    default:
      if (typeof toolName === 'string' && toolName.length > 0) {
        return `Using ${toolName}...`;
      }
      return 'Working...';
  }
}

/**
 * Start the WebSocket server that bridges browser extension <-> coach agent.
 *
 * Each WebSocket connection spawns a dedicated Claude Code subprocess.
 * Messages from the extension are forwarded to Claude's stdin.
 * Claude's stdout is streamed back over the WebSocket.
 */
export function startServe(options = {}) {
  const port = options.port || DEFAULT_PORT;
  const host = options.host || 'localhost';

  const httpServer = createServer((req, res) => {
    if (req.url === '/health') {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ status: 'ok' }));
      return;
    }
    res.writeHead(404);
    res.end();
  });

  const coachSystemPrompt = buildServeCoachPrompt();

  const wss = new WebSocketServer({ server: httpServer });
  const connections = new Map(); // ws -> { claude, id, api }
  let connectionCounter = 0;

  wss.on('connection', (ws, req) => {
    const connId = ++connectionCounter;
    const origin = req.headers.origin || 'unknown';
    console.log(`[serve] Connection #${connId} opened (origin: ${origin})`);

    // Create the typed WebSocket API for this connection
    const adapter = createWSAdapter(ws);
    const api = createServeApi(adapter);

    // Spawn a Claude Code subprocess for this connection, with coach system prompt
    const claude = spawn('claude', [
      '-p',
      '--verbose',
      '--dangerously-skip-permissions',
      '--output-format', 'stream-json',
      '--input-format', 'stream-json',
      '--system-prompt', coachSystemPrompt,
    ], {
      stdio: ['pipe', 'pipe', 'pipe'],
      env: { ...process.env },
    });

    connections.set(ws, { claude, id: connId, api });

    // Stream Claude stdout -> WebSocket
    // Filter stream-json events to only forward useful content to the extension
    let stdoutBuf = '';
    let lastStatusKey = 'idle:'; // Track last sent status to avoid duplicates

    function sendStatus(state, detail) {
      if (ws.readyState !== ws.OPEN) return;
      // Avoid sending duplicate status with same state+detail
      const key = `${state}:${detail || ''}`;
      if (key === lastStatusKey) return;
      lastStatusKey = key;
      api.send.status(detail ? { state, detail } : { state });
    }

    claude.stdout.on('data', (chunk) => {
      stdoutBuf += chunk.toString();
      // stream-json outputs newline-delimited JSON
      const lines = stdoutBuf.split('\n');
      stdoutBuf = lines.pop(); // keep incomplete line in buffer
      for (const line of lines) {
        if (!line.trim() || ws.readyState !== ws.OPEN) continue;
        try {
          const event = JSON.parse(line);

          if (event.type === 'assistant') {
            // Send idle status when assistant content arrives
            sendStatus('idle');
            // Extract text content from the assistant message
            const textParts = (event.message?.content || [])
              .filter(c => c.type === 'text')
              .map(c => c.text);
            if (textParts.length > 0) {
              api.send.assistant({
                content: textParts.join(''),
                sessionId: event.session_id,
              });
            }
          } else if (event.type === 'result') {
            sendStatus('idle');
            api.send.result({
              content: event.result || '',
              isError: event.is_error || false,
              sessionId: event.session_id,
            });
          } else if (event.type === 'tool_use') {
            // Claude is using a tool -- send working status with detail
            const toolName = event.tool?.name || event.name || '';
            sendStatus('working', describeToolUse(toolName));
          } else if (event.type === 'tool_result') {
            // Tool finished; Claude will either think again or respond
            sendStatus('thinking');
          } else if (event.type === 'system' || event.type === 'init') {
            // Processing has started, Claude is thinking
            sendStatus('thinking');
          }
          // Skip: rate_limit_event, etc.
        } catch {
          // Non-JSON line, skip
        }
      }
    });

    // Forward stderr as error messages
    claude.stderr.on('data', (chunk) => {
      const msg = chunk.toString().trim();
      console.log(`[serve] #${connId} stderr: ${msg.slice(0, 300)}`);
      if (msg && ws.readyState === ws.OPEN) {
        api.send.error({ message: msg });
      }
    });

    claude.on('exit', (code) => {
      console.log(`[serve] Claude process for connection #${connId} exited (code: ${code})`);
      if (ws.readyState === ws.OPEN) {
        api.send.agentExited({ code: code ?? 1 });
        ws.close(1000, 'Agent process exited');
      }
      connections.delete(ws);
    });

    claude.on('error', (err) => {
      console.error(`[serve] Claude process error for connection #${connId}:`, err.message);
      if (ws.readyState === ws.OPEN) {
        api.send.error({ message: `Agent error: ${err.message}` });
        ws.close(1011, 'Agent process error');
      }
      connections.delete(ws);
    });

    // Helper to write a formatted message to Claude's stdin
    function sendToClaude(formatted) {
      if (formatted === null) return;
      console.log(`[serve] #${connId} <- formatted: ${formatted.slice(0, 300)}`);
      if (!claude.stdin.destroyed) {
        sendStatus('thinking');
        const stdinMsg = JSON.stringify({
          type: 'user',
          message: { role: 'user', content: formatted },
          session_id: 'default',
        });
        claude.stdin.write(stdinMsg + '\n');
        console.log(`[serve] #${connId} wrote to claude stdin: ${stdinMsg.slice(0, 300)}`);
      } else {
        console.log(`[serve] #${connId} stdin is destroyed, cannot write`);
      }
    }

    // Register typed message handlers
    api.on.chat((data) => {
      console.log(`[serve] #${connId} <- coach:chat`);
      const formatted = formatChatMessage(data);
      sendToClaude(formatted);
    });

    api.on.elementSelection((data) => {
      console.log(`[serve] #${connId} <- coach:elementSelection`);
      const formatted = `[User selected an element]\n${formatElementSelection(data)}`;
      sendToClaude(formatted);
    });

    api.on.actionRecording((data) => {
      console.log(`[serve] #${connId} <- coach:actionRecording`);
      const formatted = `[User recorded browser actions]\n${formatActionRecording(data)}`;
      sendToClaude(formatted);
    });

    api.on.init((data) => {
      console.log(`[serve] #${connId} <- coach:init`);
      const formatted = formatInitMessage(data);
      sendToClaude(formatted);
    });

    ws.on('close', (code, reason) => {
      console.log(`[serve] Connection #${connId} closed (code: ${code})`);
      const conn = connections.get(ws);
      if (conn) {
        conn.claude.kill('SIGTERM');
        connections.delete(ws);
      }
    });

    ws.on('error', (err) => {
      console.error(`[serve] WebSocket error on connection #${connId}:`, err.message);
    });
  });

  // Ping all connections every 30s to detect stale connections
  const pingInterval = setInterval(() => {
    for (const [ws] of connections) {
      if (ws.readyState === ws.OPEN) {
        ws.ping();
      }
    }
  }, 30000);

  // Graceful shutdown
  function shutdown(signal) {
    console.log(`\n[serve] Received ${signal}, shutting down...`);
    clearInterval(pingInterval);

    for (const [ws, conn] of connections) {
      conn.claude.kill('SIGTERM');
      if (ws.readyState === ws.OPEN) {
        ws.close(1001, 'Server shutting down');
      }
    }
    connections.clear();

    wss.close(() => {
      httpServer.close(() => {
        console.log('[serve] Server stopped.');
        process.exit(0);
      });
    });

    // Force exit after 5s if graceful shutdown stalls
    setTimeout(() => process.exit(1), 5000);
  }

  process.on('SIGINT', () => shutdown('SIGINT'));
  process.on('SIGTERM', () => shutdown('SIGTERM'));

  httpServer.listen(port, host, () => {
    console.log(`[serve] WebSocket server listening on ws://${host}:${port}`);
    console.log(`[serve] Health check: http://${host}:${port}/health`);
    console.log('[serve] Press Ctrl+C to stop.');
  });

  return { httpServer, wss, shutdown };
}
