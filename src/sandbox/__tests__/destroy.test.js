import { test, describe, beforeEach, afterEach, mock } from 'node:test';
import assert from 'node:assert';
import fs from 'fs/promises';
import path from 'path';
import os from 'os';

let tmpDir;
let originalHome;

beforeEach(async () => {
  tmpDir = await fs.mkdtemp(path.join(os.tmpdir(), 'spekk-destroy-test-'));
  originalHome = process.env.HOME;
  process.env.HOME = tmpDir;
});

afterEach(async () => {
  process.env.HOME = originalHome;
  await fs.rm(tmpDir, { recursive: true, force: true });
  mock.restoreAll();
});

async function writeSandbox(name, data) {
  const spekkDir = path.join(tmpDir, '.spekk');
  await fs.mkdir(spekkDir, { recursive: true });
  let existing = {};
  try {
    existing = JSON.parse(await fs.readFile(path.join(spekkDir, 'sandboxes.json'), 'utf8'));
  } catch {}
  existing[name] = data;
  await fs.writeFile(path.join(spekkDir, 'sandboxes.json'), JSON.stringify(existing));
}

async function readSandboxes() {
  try {
    return JSON.parse(await fs.readFile(path.join(tmpDir, '.spekk', 'sandboxes.json'), 'utf8'));
  } catch {
    return {};
  }
}

describe('sandbox destroy', () => {
  test('exits with error if sandbox not found', async () => {
    let exitCode = null;
    let errorMsg = '';
    mock.method(process, 'exit', (code) => { exitCode = code; throw new Error('exit'); });
    mock.method(console, 'error', (msg) => { errorMsg = msg; });

    const q = `?t=${Date.now()}-${Math.random()}`;
    const { destroyCommand } = await import(`../destroy.js${q}`);

    await assert.rejects(() => destroyCommand(['nonexistent', '--force']), { message: 'exit' });
    assert.strictEqual(exitCode, 1);
    assert.ok(errorMsg.includes('not found'));
  });

  test('exits with error if no name provided', async () => {
    let exitCode = null;
    mock.method(process, 'exit', (code) => { exitCode = code; throw new Error('exit'); });
    mock.method(console, 'error', () => {});

    const q = `?t=${Date.now()}-${Math.random()}`;
    const { destroyCommand } = await import(`../destroy.js${q}`);

    await assert.rejects(() => destroyCommand([]), { message: 'exit' });
    assert.strictEqual(exitCode, 1);
  });
});
