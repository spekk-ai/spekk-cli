import { test, describe, beforeEach, afterEach, mock } from 'node:test';
import assert from 'node:assert';
import fs from 'fs/promises';
import path from 'path';
import os from 'os';
import { deployCommand } from '../deploy.js';

let originalHome;
let originalFetch;
let tmpDir;

beforeEach(async () => {
  tmpDir = await fs.mkdtemp(path.join(os.tmpdir(), 'spekk-deploy-test-'));
  originalHome = process.env.HOME;
  originalFetch = globalThis.fetch;
  process.env.HOME = tmpDir;
  process.env.GITHUB_TOKEN = 'test-token';

  // Mock fetch so fetchAgentClient succeeds (returns a base64-encoded Python script)
  const fakeContent = Buffer.from('#!/usr/bin/env python3\nimport websockets\n').toString('base64');
  globalThis.fetch = mock.fn(async () => ({
    ok: true,
    status: 200,
    json: async () => ({ content: fakeContent }),
  }));
});

afterEach(async () => {
  process.env.HOME = originalHome;
  globalThis.fetch = originalFetch;
  await fs.rm(tmpDir, { recursive: true, force: true });
});

/**
 * Write sandbox data directly to the filesystem under the temp HOME.
 */
async function writeSandboxFile(sandboxes) {
  const spekkDir = path.join(tmpDir, '.spekk');
  await fs.mkdir(spekkDir, { recursive: true });
  await fs.writeFile(path.join(spekkDir, 'sandboxes.json'), JSON.stringify(sandboxes, null, 2));
}

describe('sandbox deploy', () => {
  test('deployCommand exits with code 1 when sandbox not found', async () => {
    const originalExit = process.exit;
    const originalError = console.error;
    let exitCode = null;
    let errorMsg = '';

    process.exit = (code) => { exitCode = code; throw new Error('EXIT'); };
    console.error = (msg) => { errorMsg = msg; };

    try {
      await deployCommand(['nonexistent']);
    } catch (e) {
      if (e.message !== 'EXIT') throw e;
    } finally {
      process.exit = originalExit;
      console.error = originalError;
    }

    assert.strictEqual(exitCode, 1);
    assert.ok(errorMsg.includes('not found'), `Expected 'not found' in: ${errorMsg}`);
  });

  test('deployCommand exits with code 1 when no name given', async () => {
    const originalExit = process.exit;
    const originalError = console.error;
    let exitCode = null;

    process.exit = (code) => { exitCode = code; throw new Error('EXIT'); };
    console.error = () => {};

    try {
      await deployCommand([]);
    } catch (e) {
      if (e.message !== 'EXIT') throw e;
    } finally {
      process.exit = originalExit;
      console.error = originalError;
    }

    assert.strictEqual(exitCode, 1);
  });

  test('deploy module exports deployCommand function', () => {
    assert.strictEqual(typeof deployCommand, 'function');
  });

  test('deploy looks up sandbox from store and attempts SCP with correct IP', async () => {
    await writeSandboxFile({
      'test-deploy': { dropletId: 999, ip: '192.168.1.100', region: 'nyc1', status: 'active' }
    });

    const originalExit = process.exit;
    const originalError = console.error;
    const originalLog = console.log;
    let errorMsg = '';
    let logMsgs = [];

    process.exit = (code) => { throw new Error('EXIT:' + code); };
    console.error = (msg) => { errorMsg += msg; };
    console.log = (msg) => { logMsgs.push(msg); };

    try {
      await deployCommand(['test-deploy']);
    } catch (e) {
      // Expected: SCP will fail since no real server
      if (!e.message.startsWith('EXIT')) throw e;
    } finally {
      process.exit = originalExit;
      console.error = originalError;
      console.log = originalLog;
    }

    // It should have gotten past the "not found" check and attempted SCP
    const allOutput = [...logMsgs, errorMsg].join(' ');
    assert.ok(
      allOutput.includes('192.168.1.100'),
      `Expected IP 192.168.1.100 in output, got: ${allOutput}`
    );
  });
});
