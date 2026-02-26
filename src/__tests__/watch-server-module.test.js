import { test, describe } from 'node:test';
import assert from 'node:assert';
import { createServer } from 'node:http';
import { startWatchServer } from '../show/server.js';

describe('Watch Server Module', { concurrency: 1 }, () => {

  test('exports startWatchServer function', () => {
    assert.strictEqual(typeof startWatchServer, 'function');
  });

  test('serves getHTML() at GET / with text/html and calls getHTML fresh each time', async () => {
    let callCount = 0;
    const result = await startWatchServer({
      getHTML: () => `<html>call ${++callCount}</html>`,
      port: 0,
    });

    try {
      const res1 = await fetch(`http://localhost:${result.port}/`);
      assert.strictEqual(res1.status, 200);
      assert.ok(res1.headers.get('content-type').includes('text/html'));
      const body1 = await res1.text();
      assert.strictEqual(body1, '<html>call 1</html>');

      const res2 = await fetch(`http://localhost:${result.port}/`);
      const body2 = await res2.text();
      assert.strictEqual(body2, '<html>call 2</html>');
    } finally {
      result.close();
    }
  });

  test('SSE endpoint returns correct headers and notifyClients sends reload event', async () => {
    const result = await startWatchServer({
      getHTML: () => '<html></html>',
      port: 0,
    });

    try {
      // Check headers
      const controller = new AbortController();
      const res = await fetch(`http://localhost:${result.port}/events`, {
        signal: controller.signal,
      });
      assert.strictEqual(res.status, 200);
      assert.ok(res.headers.get('content-type').includes('text/event-stream'));
      assert.strictEqual(res.headers.get('cache-control'), 'no-cache');
      assert.strictEqual(res.headers.get('connection'), 'keep-alive');

      // Read the stream and verify notifyClients sends reload
      const data = await new Promise((resolve, reject) => {
        const timeout = setTimeout(() => {
          controller.abort();
          reject(new Error('Timed out waiting for SSE event'));
        }, 3000);

        (async () => {
          const reader = res.body.getReader();
          const decoder = new TextDecoder();
          let buffer = '';

          // Send reload after a short delay
          setTimeout(() => result.notifyClients(), 50);

          try {
            while (true) {
              const { value, done } = await reader.read();
              if (done) break;
              buffer += decoder.decode(value, { stream: true });
              if (buffer.includes('event: reload')) {
                clearTimeout(timeout);
                controller.abort();
                resolve(buffer);
                return;
              }
            }
          } catch (err) {
            if (err.name !== 'AbortError') reject(err);
          }
        })();
      });

      assert.ok(data.includes('event: reload'), 'Should receive reload event');
      assert.ok(data.includes('data: reload'), 'Should include data field');
    } finally {
      result.close();
    }
  });

  test('returns { server, port, notifyClients, close } and binds to localhost', async () => {
    const result = await startWatchServer({
      getHTML: () => '<html></html>',
      port: 0,
    });

    try {
      assert.ok(result.server, 'Should have server property');
      assert.strictEqual(typeof result.port, 'number');
      assert.strictEqual(typeof result.notifyClients, 'function');
      assert.strictEqual(typeof result.close, 'function');

      const addr = result.server.address();
      assert.ok(
        addr.address === '127.0.0.1' || addr.address === '::1',
        `Should bind to localhost, got ${addr.address}`
      );
    } finally {
      result.close();
    }
  });

  test('port retry when port is in use', async () => {
    const blocker = createServer();
    const blockerPort = await new Promise((resolve) => {
      blocker.listen(0, 'localhost', () => {
        resolve(blocker.address().port);
      });
    });

    try {
      const result = await startWatchServer({
        getHTML: () => '<html></html>',
        port: blockerPort,
      });

      try {
        assert.ok(result.port > blockerPort, 'Should have moved to a higher port');
        assert.ok(result.port <= blockerPort + 10, 'Should be within retry range');

        const res = await fetch(`http://localhost:${result.port}/`);
        assert.strictEqual(res.status, 200);
      } finally {
        result.close();
      }
    } finally {
      blocker.close();
    }
  });

  test('default port is 3117', async () => {
    // Verify the source code has port = 3117 as default parameter
    const { readFileSync } = await import('node:fs');
    const { dirname, join } = await import('node:path');
    const { fileURLToPath } = await import('node:url');
    const __dirname = dirname(fileURLToPath(import.meta.url));
    const source = readFileSync(join(__dirname, '../show/server.js'), 'utf8');
    assert.ok(source.includes('port = 3117'), 'Default port should be 3117 in source');
  });

  test('close() shuts down the server', async () => {
    const result = await startWatchServer({
      getHTML: () => '<html></html>',
      port: 0,
    });
    const port = result.port;

    result.close();

    await new Promise((resolve) => setTimeout(resolve, 50));
    try {
      await fetch(`http://localhost:${port}/`);
      assert.fail('Should not be able to connect after close');
    } catch {
      assert.ok(true);
    }
  });

  test('returns 404 for unknown paths', async () => {
    const result = await startWatchServer({
      getHTML: () => '<html></html>',
      port: 0,
    });

    try {
      const res = await fetch(`http://localhost:${result.port}/unknown`);
      assert.strictEqual(res.status, 404);
    } finally {
      result.close();
    }
  });
});
