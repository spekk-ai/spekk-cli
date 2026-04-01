import { test, describe, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert';
import { existsSync } from 'node:fs';
import { readFile } from 'node:fs/promises';
import { mock } from 'node:test';
import { fetchReleaseArtifacts } from '../release.js';

const FAKE_BINARY = Buffer.from([0x7f, 0x45, 0x4c, 0x46]); // ELF magic bytes
const FAKE_CLOUD_INIT = '#cloud-config\npackages:\n  - docker.io\n';

describe('fetchReleaseArtifacts', () => {
  let originalFetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    process.env.GITHUB_TOKEN = 'test-token';
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  test('fetches release, downloads both assets, returns paths and version', async () => {
    globalThis.fetch = mock.fn(async (url, opts) => {
      assert.ok(opts.headers.Authorization.includes('test-token'), 'should use GITHUB_TOKEN');

      if (url.includes('/releases/latest')) {
        return { ok: true, status: 200, json: async () => ({
          tag_name: 'v1.2.3',
          assets: [
            { name: 'sandbox', id: 1001 },
            { name: 'cloud-init.yaml', id: 1002 },
          ],
        }) };
      }
      if (url.includes('/releases/assets/1001')) {
        return { ok: true, status: 200, arrayBuffer: async () => FAKE_BINARY.buffer };
      }
      if (url.includes('/releases/assets/1002')) {
        return { ok: true, status: 200, text: async () => FAKE_CLOUD_INIT };
      }
      throw new Error(`Unexpected URL: ${url}`);
    });

    const result = await fetchReleaseArtifacts('latest');

    assert.strictEqual(result.version, 'v1.2.3');
    assert.ok(existsSync(result.binaryPath), 'binary temp file should exist');
    assert.ok(existsSync(result.cloudInitPath), 'cloud-init temp file should exist');

    const cloudInitContent = await readFile(result.cloudInitPath, 'utf-8');
    assert.strictEqual(cloudInitContent, FAKE_CLOUD_INIT);
  });

  test('exits with HTTP status when release fetch fails', async () => {
    const originalExit = process.exit;
    const originalError = console.error;
    let exitCode = null;
    let errorMsg = '';

    process.exit = (code) => { exitCode = code; throw new Error('EXIT'); };
    console.error = (msg) => { errorMsg += msg; };

    globalThis.fetch = mock.fn(async () => ({ ok: false, status: 404 }));

    try {
      await fetchReleaseArtifacts('latest');
    } catch (e) {
      if (e.message !== 'EXIT') throw e;
    } finally {
      process.exit = originalExit;
      console.error = originalError;
    }

    assert.strictEqual(exitCode, 1);
    assert.ok(errorMsg.includes('404'), 'error should include HTTP status');
  });

  test('exits when a required asset is missing from release', async () => {
    const releaseNoSandbox = {
      tag_name: 'v1.0.0',
      assets: [{ name: 'cloud-init.yaml', id: 1002 }],
    };

    const originalExit = process.exit;
    const originalError = console.error;
    let exitCode = null;
    let errorMsg = '';

    process.exit = (code) => { exitCode = code; throw new Error('EXIT'); };
    console.error = (msg) => { errorMsg += msg; };

    globalThis.fetch = mock.fn(async () => ({
      ok: true, status: 200, json: async () => releaseNoSandbox,
    }));

    try {
      await fetchReleaseArtifacts('latest');
    } catch (e) {
      if (e.message !== 'EXIT') throw e;
    } finally {
      process.exit = originalExit;
      console.error = originalError;
    }

    assert.strictEqual(exitCode, 1);
    assert.ok(errorMsg.includes('sandbox'), 'error should mention missing asset name');
  });
});
