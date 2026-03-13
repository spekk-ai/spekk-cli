import { describe, it, beforeEach, afterEach, mock } from 'node:test';
import assert from 'node:assert';

// Mock all external dependencies before importing create.js
// We use dynamic import with cache-busting to get fresh modules

describe('createSandbox', () => {
  let originalFetch, originalEnv, logs, errors;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    originalEnv = { ...process.env };
    logs = [];
    errors = [];
    const origLog = console.log;
    const origErr = console.error;
    console.log = (...args) => logs.push(args.join(' '));
    console.error = (...args) => errors.push(args.join(' '));
    console._origLog = origLog;
    console._origErr = origErr;

    // Set required env vars
    process.env.DO_API_TOKEN = 'test-token';
    process.env.AWS_ACCESS_KEY_ID = 'AKIAIOSFODNN7EXAMPLE';
    process.env.AWS_SECRET_ACCESS_KEY = 'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY';
    process.env.AWS_DEFAULT_REGION = 'us-east-1';
    process.env.GITHUB_TOKEN = 'ghp_test';
    process.env.SPEKK_AGENT_TOKEN = 'agent-token';
    process.env.SPEKK_HOST = 'https://spekk.example.com';
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    console.log = console._origLog;
    console.error = console._origErr;
    // Restore env
    for (const key of Object.keys(process.env)) {
      if (!(key in originalEnv)) delete process.env[key];
    }
    Object.assign(process.env, originalEnv);
    process.exitCode = undefined;
  });

  it('errors when required environment variables are missing', async () => {
    delete process.env.AWS_ACCESS_KEY_ID;
    delete process.env.GITHUB_TOKEN;

    // Mock fetch for DO API calls
    globalThis.fetch = mock.fn(async () => ({
      ok: true, status: 200, json: async () => ({ ssh_keys: [{ id: 1 }] })
    }));

    // We need to test the module directly - use a simpler approach
    // Just verify the env check logic
    const { createSandbox } = await import('../create.js?v=envcheck');
    await createSandbox({ name: 'test' });

    assert.strictEqual(process.exitCode, 1);
    const allOutput = errors.join(' ');
    assert.ok(allOutput.includes('AWS_ACCESS_KEY_ID'), 'Should mention missing AWS_ACCESS_KEY_ID');
    assert.ok(allOutput.includes('GITHUB_TOKEN'), 'Should mention missing GITHUB_TOKEN');
  });

  it('errors when no SSH keys are available', async () => {
    // Mock DO API to return empty SSH keys
    globalThis.fetch = mock.fn(async (url) => {
      if (url.includes('/v2/account/keys')) {
        return { ok: true, status: 200, json: async () => ({ ssh_keys: [] }) };
      }
      return { ok: true, status: 200, json: async () => ({}) };
    });

    const { createSandbox } = await import('../create.js?v=nosshkeys');
    await createSandbox({ name: 'test' });

    assert.strictEqual(process.exitCode, 1);
    assert.ok(errors.some(e => e.includes('No SSH keys found')));
  });
});
