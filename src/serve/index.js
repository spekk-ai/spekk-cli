import { WebSocketServer } from 'ws';
import { spawn } from 'node:child_process';
import { createServer } from 'node:http';
import { COACH_SYSTEM_PROMPT } from './coach-prompt.js';
import { formatMessageForClaude } from './message-formatter.js';

const DEFAULT_PORT = 3118;

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

  const wss = new WebSocketServer({ server: httpServer });
  const connections = new Map(); // ws -> { claude, id }
  let connectionCounter = 0;

  wss.on('connection', (ws, req) => {
    const connId = ++connectionCounter;
    const origin = req.headers.origin || 'unknown';
    console.log(`[serve] Connection #${connId} opened (origin: ${origin})`);

    // Spawn a Claude Code subprocess for this connection, with coach system prompt
    const claude = spawn('claude', [
      '-p',
      '--verbose',
      '--dangerously-skip-permissions',
      '--output-format', 'stream-json',
      '--input-format', 'stream-json',
      '--system-prompt', COACH_SYSTEM_PROMPT,
    ], {
      stdio: ['pipe', 'pipe', 'pipe'],
      env: { ...process.env },
    });

    connections.set(ws, { claude, id: connId });

    // Stream Claude stdout -> WebSocket
    // Filter stream-json events to only forward useful content to the extension
    let stdoutBuf = '';
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
            // Extract text content from the assistant message
            const textParts = (event.message?.content || [])
              .filter(c => c.type === 'text')
              .map(c => c.text);
            if (textParts.length > 0) {
              ws.send(JSON.stringify({
                type: 'assistant',
                content: textParts.join(''),
                session_id: event.session_id,
              }));
            }
          } else if (event.type === 'result') {
            ws.send(JSON.stringify({
              type: 'result',
              content: event.result || '',
              is_error: event.is_error || false,
              session_id: event.session_id,
            }));
          }
          // Skip: system/init, rate_limit_event, etc.
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
        ws.send(JSON.stringify({ type: 'error', message: msg }));
      }
    });

    claude.on('exit', (code) => {
      console.log(`[serve] Claude process for connection #${connId} exited (code: ${code})`);
      if (ws.readyState === ws.OPEN) {
        ws.send(JSON.stringify({ type: 'system', event: 'agent_exited', code }));
        ws.close(1000, 'Agent process exited');
      }
      connections.delete(ws);
    });

    claude.on('error', (err) => {
      console.error(`[serve] Claude process error for connection #${connId}:`, err.message);
      if (ws.readyState === ws.OPEN) {
        ws.send(JSON.stringify({ type: 'error', message: `Agent error: ${err.message}` }));
        ws.close(1011, 'Agent process error');
      }
      connections.delete(ws);
    });

    // Extension messages -> Claude stdin (formatted for readability)
    ws.on('message', (data) => {
      const raw = data.toString();
      console.log(`[serve] #${connId} ← raw: ${raw.slice(0, 300)}`);
      const formatted = formatMessageForClaude(raw);
      console.log(`[serve] #${connId} ← formatted: ${formatted ? formatted.slice(0, 300) : '(null, skipped)'}`);

      // null means the message should not be forwarded (e.g., ping)
      if (formatted === null) {
        return;
      }

      if (!claude.stdin.destroyed) {
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
