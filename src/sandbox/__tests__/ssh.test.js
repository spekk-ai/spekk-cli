import { test, describe, beforeEach, afterEach, mock } from 'node:test';
import assert from 'node:assert';
import fs from 'fs/promises';
import path from 'path';
import os from 'os';

let tmpDir;
let originalHome;

beforeEach(async () => {
  tmpDir = await fs.mkdtemp(path.join(os.tmpdir(), 'spekk-ssh-test-'));
  originalHome = process.env.HOME;
  process.env.HOME = tmpDir;
});

afterEach(async () => {
  process.env.HOME = originalHome;
  await fs.rm(tmpDir, { recursive: true, force: true });
  mock.restoreAll();
});

describe('sandbox ssh', () => {
  test('exits with error if sandbox name not found', async () => {
    let exitCode = null;
    let errorMsg = '';
    mock.method(process, 'exit', (code) => { exitCode = code; throw new Error('exit'); });
    mock.method(console, 'error', (msg) => { errorMsg = msg; });

    const uniqueQ = `?t=${Date.now()}-${Math.random()}`;
    const { sshCommand } = await import(`../ssh.js${uniqueQ}`);

    await assert.rejects(() => sshCommand(['nonexistent']), { message: 'exit' });
    assert.strictEqual(exitCode, 1);
    assert.ok(errorMsg.includes('not found'));
  });

  test('exits with error if no name provided', async () => {
    let exitCode = null;
    mock.method(process, 'exit', (code) => { exitCode = code; throw new Error('exit'); });
    mock.method(console, 'error', () => {});

    const uniqueQ = `?t=${Date.now()}-${Math.random()}`;
    const { sshCommand } = await import(`../ssh.js${uniqueQ}`);

    await assert.rejects(() => sshCommand([]), { message: 'exit' });
    assert.strictEqual(exitCode, 1);
  });
});
