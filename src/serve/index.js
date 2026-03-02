import { WebSocketServer } from 'ws';
import { spawn } from 'node:child_process';
import { createServer } from 'node:http';
import { COACH_SYSTEM_PROMPT } from './coach-prompt.js';
import { formatMessageForClaude } from './message-formatter.js';

const DEFAULT_PORT = 3118;
const DEFAULT_IDLE_TIMEOUT = 30 * 60; // 30 minutes in seconds

/**
 * Spawn a Claude Code subprocess and wire its stdout/stderr to a WebSocket.
 * Returns the child process handle.
 */
function spawnClaudeForConnection(ws, connId) {
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

  // Stream Claude stdout -> WebSocket
  let stdoutBuf = '';

  claude.stdout.on('data', (chunk) => {
    stdoutBuf += chunk.toString();
    const lines = stdoutBuf.split('\n');
    stdoutBuf = lines.pop();
    for (const line of lines) {
      if (!line.trim() || ws.readyState !== ws.OPEN) continue;
      try {
        const event = JSON.parse(line);
        if (event.type === 'assistant') {
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
    console.log('[serve] #' + connId + ' stderr: ' + msg.slice(0, 300));
    if (msg && ws.readyState === ws.OPEN) {
      ws.send(JSON.stringify({ type: 'error', message: msg }));
    }
  });

  return claude;
}

/**
 * Start the WebSocket server that bridges browser extension <-> coach agent.
 *
 * Enforces a single active connection at a time per port. When a new client
 * connects while an existing session is active, the server sends a
 * connection_locked message and waits for a force_takeover request.
 *
 * Idle connections are automatically timed out after idleTimeout seconds.
 */
export function startServe(options = {}) {
  const port = options.port || DEFAULT_PORT;
  const host = options.host || 'localhost';
  const idleTimeout = options.idleTimeout ?? DEFAULT_IDLE_TIMEOUT;

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

  // Single active session state
  let activeSession = null; // { ws, claude, id, connectedAt, lastActivityAt }
  let connectionCounter = 0;

  // Pending connections waiting for takeover decision (ws -> { id })
  const pendingConnections = new Map();

  /**
   * Clean up the active session: kill Claude, close WebSocket, clear state.
   */
  function cleanupActiveSession() {
    if (!activeSession) return;
    const { ws, claude, id } = activeSession;
    console.log('[serve] Cleaning up active session #' + id);
    claude.kill('SIGTERM');
    if (ws.readyState === ws.OPEN || ws.readyState === ws.CONNECTING) {
      ws.close(1000, 'Session ended');
    }
    activeSession = null;
  }

  /**
   * Promote a WebSocket connection to the active session by spawning Claude.
   */
  function promoteToActive(ws, connId) {
    const now = Date.now();
    const claude = spawnClaudeForConnection(ws, connId);

    activeSession = {
      ws,
      claude,
      id: connId,
      connectedAt: new Date().toISOString(),
      lastActivityAt: now,
    };

    pendingConnections.delete(ws);

    claude.on('exit', (code) => {
      console.log('[serve] Claude process for connection #' + connId + ' exited (code: ' + code + ')');
      if (ws.readyState === ws.OPEN) {
        ws.send(JSON.stringify({ type: 'system', event: 'agent_exited', code }));
        ws.close(1000, 'Agent process exited');
      }
      if (activeSession && activeSession.ws === ws) {
        activeSession = null;
      }
    });

    claude.on('error', (err) => {
      console.error('[serve] Claude process error for connection #' + connId + ':', err.message);
      if (ws.readyState === ws.OPEN) {
        ws.send(JSON.stringify({ type: 'error', message: 'Agent error: ' + err.message }));
        ws.close(1011, 'Agent process error');
      }
      if (activeSession && activeSession.ws === ws) {
        activeSession = null;
      }
    });

    console.log('[serve] Connection #' + connId + ' promoted to active session');
  }

  /**
   * Wire up the standard message handler for the active session's WebSocket.
   */
  function wireActiveMessageHandler(ws, connId) {
    ws.on('message', (data) => {
      const raw = data.toString();
      console.log('[serve] #' + connId + ' <- raw: ' + raw.slice(0, 300));

      // Update idle tracker on every message from the active client
      if (activeSession && activeSession.ws === ws) {
        activeSession.lastActivityAt = Date.now();
      }

      const formatted = formatMessageForClaude(raw);
      console.log('[serve] #' + connId + ' <- formatted: ' + (formatted ? formatted.slice(0, 300) : '(null, skipped)'));

      if (formatted === null) {
        return;
      }

      if (activeSession && activeSession.ws === ws && !activeSession.claude.stdin.destroyed) {
        const stdinMsg = JSON.stringify({
          type: 'user',
          message: { role: 'user', content: formatted },
          session_id: 'default',
        });
        activeSession.claude.stdin.write(stdinMsg + '\n');
        console.log('[serve] #' + connId + ' wrote to claude stdin: ' + stdinMsg.slice(0, 300));
      } else {
        console.log('[serve] #' + connId + ' stdin is destroyed or session mismatch, cannot write');
      }
    });

    ws.on('close', (code) => {
      console.log('[serve] Connection #' + connId + ' closed (code: ' + code + ')');
      if (activeSession && activeSession.ws === ws) {
        activeSession.claude.kill('SIGTERM');
        activeSession = null;
      }
    });

    ws.on('error', (err) => {
      console.error('[serve] WebSocket error on connection #' + connId + ':', err.message);
    });
  }

  wss.on('connection', (ws, req) => {
    const connId = ++connectionCounter;
    const origin = req.headers.origin || 'unknown';
    console.log('[serve] Connection #' + connId + ' opened (origin: ' + origin + ')');

    // Check if there is already an active session with an open WebSocket
    if (activeSession && activeSession.ws.readyState === ws.OPEN) {
      // Slot is occupied - send connection_locked to the new client
      const idleForSeconds = Math.floor((Date.now() - activeSession.lastActivityAt) / 1000);
      console.log('[serve] Connection #' + connId + ' blocked - active session #' + activeSession.id + ' (idle ' + idleForSeconds + 's)');

      ws.send(JSON.stringify({
        type: 'connection_locked',
        active_since: activeSession.connectedAt,
        idle_for: idleForSeconds,
      }));

      // Track this as a pending connection waiting for force_takeover
      pendingConnections.set(ws, { id: connId });

      // Listen for force_takeover from the pending client
      ws.on('message', (data) => {
        const raw = data.toString();
        console.log('[serve] #' + connId + ' (pending) <- ' + raw.slice(0, 300));

        let parsed;
        try {
          parsed = JSON.parse(raw);
        } catch {
          return;
        }

        if (parsed.type === 'force_takeover') {
          console.log('[serve] Connection #' + connId + ' requested force takeover');
          cleanupActiveSession();
          ws.removeAllListeners('message');
          promoteToActive(ws, connId);
          wireActiveMessageHandler(ws, connId);
        }
      });

      ws.on('close', (code) => {
        console.log('[serve] Pending connection #' + connId + ' closed (code: ' + code + ')');
        pendingConnections.delete(ws);
      });

      ws.on('error', (err) => {
        console.error('[serve] WebSocket error on pending connection #' + connId + ':', err.message);
      });

      return; // Do not spawn Claude for locked-out connections
    }

    // No active session (or stale session with closed WebSocket)
    if (activeSession) {
      cleanupActiveSession();
    }
    promoteToActive(ws, connId);
    wireActiveMessageHandler(ws, connId);
  });

  // Idle timeout check - runs every 60 seconds
  const idleCheckInterval = setInterval(() => {
    if (!activeSession || idleTimeout <= 0) return;

    const idleForSeconds = Math.floor((Date.now() - activeSession.lastActivityAt) / 1000);
    if (idleForSeconds >= idleTimeout) {
      console.log('[serve] Active session #' + activeSession.id + ' timed out after ' + idleForSeconds + 's idle');
      const { ws } = activeSession;
      if (ws.readyState === ws.OPEN) {
        ws.send(JSON.stringify({ type: 'session_timeout' }));
      }
      cleanupActiveSession();
    }
  }, 60000);

  // Ping all open connections every 30s to detect stale connections
  const pingInterval = setInterval(() => {
    if (activeSession && activeSession.ws.readyState === activeSession.ws.OPEN) {
      activeSession.ws.ping();
    }
    for (const [ws] of pendingConnections) {
      if (ws.readyState === ws.OPEN) {
        ws.ping();
      }
    }
  }, 30000);

  // Graceful shutdown
  function shutdown(signal) {
    console.log('\n[serve] Received ' + signal + ', shutting down...');
    clearInterval(pingInterval);
    clearInterval(idleCheckInterval);

    if (activeSession) {
      activeSession.claude.kill('SIGTERM');
      if (activeSession.ws.readyState === activeSession.ws.OPEN) {
        activeSession.ws.close(1001, 'Server shutting down');
      }
      activeSession = null;
    }

    for (const [ws] of pendingConnections) {
      if (ws.readyState === ws.OPEN) {
        ws.close(1001, 'Server shutting down');
      }
    }
    pendingConnections.clear();

    wss.close(() => {
      httpServer.close(() => {
        console.log('[serve] Server stopped.');
        process.exit(0);
      });
    });

    setTimeout(() => process.exit(1), 5000);
  }

  process.on('SIGINT', () => shutdown('SIGINT'));
  process.on('SIGTERM', () => shutdown('SIGTERM'));

  httpServer.listen(port, host, () => {
    console.log('[serve] WebSocket server listening on ws://' + host + ':' + port);
    console.log('[serve] Health check: http://' + host + ':' + port + '/health');
    console.log('[serve] Idle timeout: ' + idleTimeout + 's');
    console.log('[serve] Press Ctrl+C to stop.');
  });

  return { httpServer, wss, shutdown };
}
