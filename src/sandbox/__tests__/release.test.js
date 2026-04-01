import { test, describe, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert';
import { existsSync } from 'node:fs';
import { readFile } from 'node:fs/promises';
import { mock } from 'node:test';
import { fetchReleaseArtifacts } from '../release.js';

const FAKE_BINARY = Buffer.from([0x7f, 0x45, 0x4c, 0x46]); // ELF magic bytes
const FAKE_CLOUD_INIT = '#cloud-config\npackages:\n  - docker.io\n';

function makeFakeRelease(tagName = 'v1.2.3') {
  return {
    tag_name: tagName,
    assets: [
      {
        name: 'sandbox',
        browser_download_url: `https://github.com/spekk-ai/spekk-app/releases/download/${tagName}/sandbox`,
      },
      {
        name: 'cloud-init.yaml',
        browser_download_url: `https://github.com/spekk-ai/spekk-app/releases/download/${tagName}/cloud-init.yaml`,
      },
    ],
  };
}

describe('fetchReleaseArtifacts', () => {
  let originalFetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    process.env.GITHUB_TOKEN = 'test-token';
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  test('fetches latest release and downloads both assets, returns paths and version', async () => {
    const fakeRelease = makeFakeRelease('v1.2.3');

    globalThis.fetch = mock.fn(async (url, opts) => {
      assert.ok(opts.headers.Authorization.includes('test-token'), 'should use GITHUB_TOKEN');

      if (url.includes('/releases/latest')) {
        return { ok: true, status: 200, json: async () => fakeRelease };
      }
      if (url.includes('/sandbox')) {
        return { ok: true, status: 200, arrayBuffer: async () => FAKE_BINARY.buffer };
      }
      if (url.includes('/cloud-init.yaml')) {
        return { ok: true, status: 200, text: async () => FAKE_CLOUD_INIT };
      }
      throw new Error(`Unexpected URL: ${url}`);
    });

    const result = await fetchReleaseArtifacts('latest');

    assert.strictEqual(result.version, 'v1.2.3');
    assert.ok(result.binaryPath, 'should return binaryPath');
    assert.ok(result.cloudInitPath, 'should return cloudInitPath');
    assert.ok(existsSync(result.binaryPath), 'binary temp file should exist');
    assert.ok(existsSync(result.cloudInitPath), 'cloud-init temp file should exist');

    const cloudInitContent = await readFile(result.cloudInitPath, 'utf-8');
    assert.strictEqual(cloudInitContent, FAKE_CLOUD_INIT);
  });

  test('uses /releases/tags/{tag} URL when tag is not "latest"', async () => {
    const fakeRelease = makeFakeRelease('v0.9.0');
    const urls = [];

    globalThis.fetch = mock.fn(async (url) => {
      urls.push(url);
      if (url.includes('/releases/tags/')) {
        return { ok: true, status: 200, json: async () => fakeRelease };
      }
      if (url.includes('/sandbox')) {
        return { ok: true, status: 200, arrayBuffer: async () => FAKE_BINARY.buffer };
      }
      if (url.includes('/cloud-init.yaml')) {
        return { ok: true, status: 200, text: async () => FAKE_CLOUD_INIT };
      }
      throw new Error(`Unexpected URL: ${url}`);
    });

    const result = await fetchReleaseArtifacts('v0.9.0');

    assert.ok(urls[0].includes('/releases/tags/v0.9.0'), 'should use tags URL for specific version');
    assert.strictEqual(result.version, 'v0.9.0');
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

  test('exits when sandbox binary asset is missing from release', async () => {
    const releaseNoSandbox = {
      tag_name: 'v1.0.0',
      assets: [{ name: 'cloud-init.yaml', browser_download_url: 'https://example.com/cloud-init.yaml' }],
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

  test('exits when cloud-init.yaml asset is missing from release', async () => {
    const releaseNoCloudInit = {
      tag_name: 'v1.0.0',
      assets: [{ name: 'sandbox', browser_download_url: 'https://example.com/sandbox' }],
    };

    const originalExit = process.exit;
    const originalError = console.error;
    let exitCode = null;
    let errorMsg = '';

    process.exit = (code) => { exitCode = code; throw new Error('EXIT'); };
    console.error = (msg) => { errorMsg += msg; };

    globalThis.fetch = mock.fn(async () => ({
      ok: true, status: 200, json: async () => releaseNoCloudInit,
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
    assert.ok(errorMsg.includes('cloud-init.yaml'), 'error should mention missing asset name');
  });
});
