import { test, describe, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert';
import { existsSync } from 'node:fs';
import { getTemplatePath, readTemplate, renderCloudInit, fetchAgentClient } from '../templates.js';
import { mock } from 'node:test';

describe('sandbox bundled templates', () => {
  test('getTemplatePath returns absolute paths to existing template files', () => {
    const cloudInitPath = getTemplatePath('cloud-init.yaml');

    assert.ok(cloudInitPath.startsWith('/'), 'path should be absolute');
    assert.ok(existsSync(cloudInitPath), 'cloud-init.yaml should exist');
  });

  test('templates directory contains only cloud-init.yaml', () => {
    const cloudInitPath = getTemplatePath('cloud-init.yaml');
    const agentClientPath = getTemplatePath('agent-client.py');

    assert.ok(existsSync(cloudInitPath), 'cloud-init.yaml should exist');
    assert.ok(!existsSync(agentClientPath), 'agent-client.py should NOT exist');
  });

  test('readTemplate returns file contents as string', async () => {
    const content = await readTemplate('cloud-init.yaml');
    assert.strictEqual(typeof content, 'string');
    assert.ok(content.includes('#cloud-config'), 'should contain cloud-config header');
  });

  test('cloud-init.yaml contains required provisioning components', async () => {
    const content = await readTemplate('cloud-init.yaml');
    assert.ok(content.includes('docker'), 'should install Docker');
    assert.ok(content.includes('nodejs') || content.includes('node'), 'should install Node.js');
    assert.ok(content.includes('git'), 'should install git');
    assert.ok(content.includes('gh'), 'should install GitHub CLI');
    assert.ok(content.includes('claude'), 'should install Claude Code CLI');
    assert.ok(content.includes('spekk-agent'), 'should set up systemd unit');
    assert.ok(content.includes('agent'), 'should reference agent user');
    assert.ok(content.includes('/opt/spekk/.provisioned'), 'should create provisioned marker');
    assert.ok(content.includes('ssh-ed25519 AAAA... your-key-here'), 'should have SSH key placeholder');
  });

  test('renderCloudInit substitutes SSH public key', async () => {
    const testKey = 'ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey user@host';
    const rendered = await renderCloudInit(testKey);
    assert.ok(rendered.includes(testKey), 'should contain the substituted key');
    assert.ok(!rendered.includes('your-key-here'), 'should not contain the placeholder');
  });
});

describe('fetchAgentClient', () => {
  let originalFetch;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    process.env.GITHUB_TOKEN = 'test-token';
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  test('fetches agent-client.py from GitHub, decodes base64, returns temp path', async () => {
    const pythonContent = '#!/usr/bin/env python3\nimport websockets\n';
    const base64Content = Buffer.from(pythonContent).toString('base64');

    globalThis.fetch = mock.fn(async (url, opts) => {
      assert.ok(url.includes('spekk-ai/spekk-app'), 'should fetch from spekk-app repo');
      assert.ok(url.includes('agent-client.py'), 'should fetch agent-client.py');
      assert.ok(opts.headers.Authorization.includes('test-token'), 'should use GITHUB_TOKEN');
      return {
        ok: true,
        status: 200,
        json: async () => ({ content: base64Content }),
      };
    });

    const tmpPath = await fetchAgentClient();
    assert.ok(tmpPath.endsWith('.py'), 'should return a .py temp file path');
    assert.ok(existsSync(tmpPath), 'temp file should exist');

    const { readFile } = await import('node:fs/promises');
    const written = await readFile(tmpPath, 'utf-8');
    assert.strictEqual(written, pythonContent, 'should decode base64 content correctly');

    // Cleanup
    const { unlink } = await import('node:fs/promises');
    await unlink(tmpPath);
  });

  test('exits on fetch failure', async () => {
    const originalExit = process.exit;
    const originalError = console.error;
    let exitCode = null;
    let errorMsg = '';

    process.exit = (code) => { exitCode = code; throw new Error('EXIT'); };
    console.error = (msg) => { errorMsg = msg; };

    globalThis.fetch = mock.fn(async () => ({
      ok: false,
      status: 404,
    }));

    try {
      await fetchAgentClient();
    } catch (e) {
      if (e.message !== 'EXIT') throw e;
    } finally {
      process.exit = originalExit;
      console.error = originalError;
    }

    assert.strictEqual(exitCode, 1);
    assert.ok(errorMsg.includes('404'), 'should include HTTP status in error');
  });
});
