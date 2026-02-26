import { test, describe } from 'node:test';
import assert from 'node:assert';
import { mkdirSync, writeFileSync, rmSync, existsSync, appendFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import http from 'node:http';
import { startWatchServer } from '../show/server.js';
import { watchSpecs } from '../show/watcher.js';
import { resolveOpenUrl } from '../show/cli.js';

function fetch(url) {
  return new Promise((resolve, reject) => {
    http.get(url, (res) => {
      let data = '';
      res.on('data', (chunk) => { data += chunk; });
      res.on('end', () => resolve({ status: res.statusCode, body: data, headers: res.headers }));
    }).on('error', reject);
  });
}

describe('Watch mode integration', () => {
  const testDir = join(tmpdir(), `spekk-watch-integration-${Date.now()}`);
  const specsDir = join(testDir, 'specs', 'test-spec', 'assertions');

  function setup() {
    mkdirSync(specsDir, { recursive: true });
    writeFileSync(join(testDir, 'specs', 'test-spec', 'test-spec.md'), `---
id: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
---

# Test Spec

A test specification.`);
    writeFileSync(join(specsDir, 'test-assertion.md'), `---
id: test-assertion
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Test Assertion

A test assertion.`);
  }

  function cleanup() {
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
  }

  test('server starts, file change triggers SSE reload, shuts down cleanly', async () => {
    cleanup();
    setup();

    let callCount = 0;
    function getHTML() {
      callCount++;
      return `<html><body><h1>Spec Explorer (render ${callCount})</h1></body></html>`;
    }

    // Use port 0 to let the OS pick an available port
    const { port, notifyClients, close } = await startWatchServer({ getHTML, port: 0 });

    try {
      // 1. Server starts and serves HTML
      const response = await fetch(`http://localhost:${port}/`);
      assert.strictEqual(response.status, 200);
      assert.ok(response.body.includes('Spec Explorer'), 'HTML should contain spec content');
      assert.ok(response.headers['content-type'].includes('text/html'), 'Content-Type should be text/html');

      // 2. SSE endpoint is available
      const sseReloadReceived = new Promise((resolve, reject) => {
        const timeout = setTimeout(() => reject(new Error('SSE reload not received within 3s')), 3000);
        const req = http.get(`http://localhost:${port}/events`, (res) => {
          assert.strictEqual(res.statusCode, 200);
          assert.ok(res.headers['content-type'].includes('text/event-stream'), 'SSE Content-Type');

          let buf = '';
          res.on('data', (chunk) => {
            buf += chunk.toString();
            if (buf.includes('event: reload')) {
              clearTimeout(timeout);
              req.destroy();
              resolve();
            }
          });
        });
        req.on('error', (err) => {
          // Ignore ECONNRESET from our intentional destroy
          if (err.code !== 'ECONNRESET') reject(err);
        });
      });

      // 3. File change triggers reload via watcher + notifyClients
      const stopWatcher = watchSpecs(join(testDir, 'specs'), () => {
        notifyClients();
      });

      // Give the SSE connection a moment to establish, then modify a spec file
      await new Promise(r => setTimeout(r, 100));
      appendFileSync(join(specsDir, 'test-assertion.md'), '\nUpdated content.');

      await sseReloadReceived;

      // Cleanup watcher
      stopWatcher();
    } finally {
      // 4. Server shuts down cleanly
      close();
    }

    // Verify server is closed by attempting a connection (should fail)
    await new Promise(r => setTimeout(r, 100));
    try {
      await fetch(`http://localhost:${port}/`);
      assert.fail('Server should be closed');
    } catch (err) {
      assert.ok(err.code === 'ECONNREFUSED' || err.code === 'ECONNRESET',
        `Expected ECONNREFUSED or ECONNRESET, got ${err.code}`);
    }

    cleanup();
  });
});

describe('resolveOpenUrl', () => {
  test('wraps file paths with file:// prefix', () => {
    assert.strictEqual(resolveOpenUrl('/home/user/.spekk/index.html'), 'file:///home/user/.spekk/index.html');
    assert.strictEqual(resolveOpenUrl('/tmp/test.html'), 'file:///tmp/test.html');
  });

  test('passes http:// URLs through unchanged', () => {
    assert.strictEqual(resolveOpenUrl('http://localhost:3117'), 'http://localhost:3117');
    assert.strictEqual(resolveOpenUrl('http://localhost:0/'), 'http://localhost:0/');
  });

  test('passes https:// URLs through unchanged', () => {
    assert.strictEqual(resolveOpenUrl('https://example.com'), 'https://example.com');
  });
});
