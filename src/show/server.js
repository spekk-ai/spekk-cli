import { createServer } from 'node:http';

/**
 * Start a watch server that serves HTML and pushes live-reload events via SSE.
 *
 * @param {Object} options
 * @param {() => string} options.getHTML - Function that returns HTML string (called fresh on every GET /)
 * @param {number} [options.port=3117] - Port to bind to (retries up to 10 times if in use)
 * @param {() => boolean} [options.isDirty] - Optional callback; if it returns true when a new SSE client connects, an immediate reload event is sent
 * @returns {Promise<{ server: import('node:http').Server, port: number, notifyClients: () => void, close: () => void }>}
 */
export async function startWatchServer({ getHTML, port = 3117, isDirty }) {
  const sseClients = new Set();

  function notifyClients() {
    for (const res of sseClients) {
      res.write('event: reload\ndata: reload\n\n');
    }
  }

  const server = createServer((req, res) => {
    if (req.method === 'GET' && req.url === '/events') {
      res.writeHead(200, {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache',
        'Connection': 'keep-alive',
      });
      res.write('\n');
      sseClients.add(res);
      if (isDirty && isDirty()) {
        res.write('event: reload\ndata: reload\n\n');
      }
      req.on('close', () => {
        sseClients.delete(res);
      });
      return;
    }

    if (req.method === 'GET' && (req.url === '/' || req.url === '')) {
      const html = getHTML();
      res.writeHead(200, { 'Content-Type': 'text/html' });
      res.end(html);
      return;
    }

    res.writeHead(404);
    res.end('Not Found');
  });

  const actualPort = await listenWithRetry(server, port, 10);

  function close() {
    for (const client of sseClients) {
      client.end();
    }
    sseClients.clear();
    server.close();
  }

  return { server, port: actualPort, notifyClients, close };
}

/**
 * Try to listen on the given port, retrying on the next port up to maxRetries times.
 */
function listenWithRetry(server, port, maxRetries) {
  return new Promise((resolve, reject) => {
    let attempts = 0;
    let currentPort = port;

    function tryListen() {
      server.once('error', onError);
      server.listen(currentPort, '127.0.0.1', () => {
        server.removeListener('error', onError);
        resolve(server.address().port);
      });
    }

    function onError(err) {
      server.removeListener('error', onError);
      if (err.code === 'EADDRINUSE' && attempts < maxRetries) {
        attempts++;
        currentPort++;
        tryListen();
      } else {
        reject(err);
      }
    }

    tryListen();
  });
}
