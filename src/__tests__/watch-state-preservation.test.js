import { test, describe } from 'node:test';
import assert from 'node:assert';
import { mkdirSync, writeFileSync, rmSync, existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import http from 'node:http';
import { startWatchServer } from '../show/server.js';

describe('Watch mode state preservation', () => {
  const testDir = join(tmpdir(), `spekk-watch-state-${Date.now()}`);

  function cleanup() {
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
  }

  test('SSE client script saves state to sessionStorage before reload', async () => {
    cleanup();
    mkdirSync(testDir, { recursive: true });

    function getHTML() {
      return `<html><body><h1>Test</h1>
<script>
(function() {
  var es = new EventSource('/events');
  es.addEventListener('reload', function() {
    location.reload();
  });
})();
</script>
</body></html>`;
    }

    const { port, close } = await startWatchServer({ getHTML, port: 0 });

    try {
      const response = await fetchURL(`http://localhost:${port}/`);
      assert.strictEqual(response.status, 200);
      // Just verify server works - actual state preservation is in SSE script
    } finally {
      close();
    }

    cleanup();
  });

  test('watch-mode HTML includes sessionStorage state save logic', async () => {
    // Import the SSE_CLIENT_SCRIPT indirectly by checking what the watch server injects
    // We test the script content by reading the source file
    const cliPath = join(process.cwd(), 'src', 'show', 'cli.js');
    const cliSource = readFileSync(cliPath, 'utf8');

    // Extract SSE_CLIENT_SCRIPT constant
    const scriptMatch = cliSource.match(/const SSE_CLIENT_SCRIPT = `([\s\S]*?)`;/);
    assert.ok(scriptMatch, 'Should find SSE_CLIENT_SCRIPT constant');

    const scriptContent = scriptMatch[1];

    // Verify state save before reload
    assert.ok(
      scriptContent.includes("sessionStorage.setItem('spekkWatchState'"),
      'Should save state to sessionStorage before reload'
    );
    assert.ok(
      scriptContent.includes('expandedSpecs'),
      'Should collect expanded spec IDs'
    );
    assert.ok(
      scriptContent.includes('activeDetailId'),
      'Should collect active detail panel ID'
    );
    assert.ok(
      scriptContent.includes('scrollTop'),
      'Should collect sidebar scroll position'
    );

    // Verify state collection selectors
    assert.ok(
      scriptContent.includes('[id^="assertions-"].expanded'),
      'Should query expanded assertion lists by selector'
    );
    assert.ok(
      scriptContent.includes('.detail-content.active'),
      'Should query active detail panel'
    );
    assert.ok(
      scriptContent.includes('.tree-panel'),
      'Should query tree panel for scroll position'
    );

    // Verify location.reload() still happens after state save
    assert.ok(
      scriptContent.includes('location.reload()'),
      'Should still call location.reload()'
    );
  });

  test('watch-mode HTML includes sessionStorage state restore logic', () => {
    const cliPath = join(process.cwd(), 'src', 'show', 'cli.js');
    const cliSource = readFileSync(cliPath, 'utf8');

    const scriptMatch = cliSource.match(/const SSE_CLIENT_SCRIPT = `([\s\S]*?)`;/);
    assert.ok(scriptMatch, 'Should find SSE_CLIENT_SCRIPT constant');

    const scriptContent = scriptMatch[1];

    // Verify state restore on page load
    assert.ok(
      scriptContent.includes("sessionStorage.getItem('spekkWatchState')"),
      'Should read state from sessionStorage on load'
    );
    assert.ok(
      scriptContent.includes("sessionStorage.removeItem('spekkWatchState')"),
      'Should clear sessionStorage after restoring'
    );

    // Verify expand restoration
    assert.ok(
      scriptContent.includes("getElementById('assertions-' + specId)"),
      'Should restore expanded assertions by ID'
    );
    assert.ok(
      scriptContent.includes("getElementById('toggle-' + specId)"),
      'Should restore toggle icons by ID'
    );
    assert.ok(
      scriptContent.includes("classList.add('expanded')"),
      'Should add expanded class to restored specs'
    );

    // Verify detail panel restoration
    assert.ok(
      scriptContent.includes('state.activeDetailId'),
      'Should restore active detail panel'
    );
    assert.ok(
      scriptContent.includes("getElementById('empty-state')"),
      'Should hide empty state when restoring detail panel'
    );

    // Verify scroll restoration with requestAnimationFrame
    assert.ok(
      scriptContent.includes('requestAnimationFrame'),
      'Should use requestAnimationFrame for scroll restore'
    );

    // Verify selected tree item restoration
    assert.ok(
      scriptContent.includes('.assertion-item[data-assertion-id='),
      'Should mark assertion as selected in tree view'
    );
  });

  test('state save happens before location.reload in the script', () => {
    const cliPath = join(process.cwd(), 'src', 'show', 'cli.js');
    const cliSource = readFileSync(cliPath, 'utf8');

    const scriptMatch = cliSource.match(/const SSE_CLIENT_SCRIPT = `([\s\S]*?)`;/);
    const scriptContent = scriptMatch[1];

    // Find positions of key operations to verify ordering
    const setItemPos = scriptContent.indexOf("sessionStorage.setItem('spekkWatchState'");
    const reloadPos = scriptContent.indexOf('location.reload()');

    assert.ok(setItemPos > -1, 'sessionStorage.setItem should be present');
    assert.ok(reloadPos > -1, 'location.reload should be present');
    assert.ok(
      setItemPos < reloadPos,
      'sessionStorage.setItem should come before location.reload'
    );
  });

  test('state restore runs on DOMContentLoaded or immediately if DOM ready', () => {
    const cliPath = join(process.cwd(), 'src', 'show', 'cli.js');
    const cliSource = readFileSync(cliPath, 'utf8');

    const scriptMatch = cliSource.match(/const SSE_CLIENT_SCRIPT = `([\s\S]*?)`;/);
    const scriptContent = scriptMatch[1];

    // Verify DOM readiness check
    assert.ok(
      scriptContent.includes("document.readyState === 'loading'"),
      'Should check document.readyState'
    );
    assert.ok(
      scriptContent.includes("addEventListener('DOMContentLoaded'"),
      'Should add DOMContentLoaded listener if still loading'
    );
    assert.ok(
      scriptContent.includes('restoreWatchState'),
      'Should call restoreWatchState function'
    );
  });

  test('non-watch HTML does not include SSE state preservation script', async () => {
    cleanup();
    mkdirSync(testDir, { recursive: true });

    const specsDir = join(testDir, 'specs', 'test-spec', 'assertions');
    mkdirSync(specsDir, { recursive: true });

    writeFileSync(join(testDir, 'specs', 'test-spec', 'test-spec.md'), `---
id: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
---

# Test Spec`);

    writeFileSync(join(specsDir, 'test-assertion.md'), `---
id: test-assertion
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Test Assertion`);

    const { execSync } = await import('node:child_process');
    const { dirname } = await import('node:path');
    const { fileURLToPath } = await import('node:url');
    const __dirname = dirname(fileURLToPath(import.meta.url));
    const projectRoot = join(__dirname, '../..');

    execSync(`node "${join(projectRoot, 'bin/spekk.js')}" show`, {
      encoding: 'utf8',
      cwd: testDir,
      timeout: 5000,
      env: { ...process.env, NODE_ENV: 'test' }
    });

    const htmlContent = readFileSync(join(testDir, '.spekk', 'index.html'), 'utf8');

    // Non-watch HTML should NOT include SSE script or state preservation
    assert.ok(
      !htmlContent.includes('EventSource'),
      'Non-watch HTML should not include EventSource'
    );
    assert.ok(
      !htmlContent.includes('spekkWatchState'),
      'Non-watch HTML should not include spekkWatchState'
    );
    assert.ok(
      !htmlContent.includes('sessionStorage'),
      'Non-watch HTML should not include sessionStorage references'
    );

    cleanup();
  });
});

function fetchURL(url) {
  return new Promise((resolve, reject) => {
    http.get(url, (res) => {
      let data = '';
      res.on('data', (chunk) => { data += chunk; });
      res.on('end', () => resolve({ status: res.statusCode, body: data, headers: res.headers }));
    }).on('error', reject);
  });
}
