import { test, describe, beforeEach, afterEach, mock } from 'node:test';
import assert from 'node:assert';
import fs from 'fs/promises';
import path from 'path';
import os from 'os';

let tmpDir;
let originalHome;

beforeEach(async () => {
  tmpDir = await fs.mkdtemp(path.join(os.tmpdir(), 'spekk-list-test-'));
  originalHome = process.env.HOME;
  process.env.HOME = tmpDir;
});

afterEach(async () => {
  process.env.HOME = originalHome;
  await fs.rm(tmpDir, { recursive: true, force: true });
  mock.restoreAll();
});

async function importList() {
  const q = `?t=${Date.now()}-${Math.random()}`;
  return import(`../list.js${q}`);
}

describe('sandbox list', () => {
  test('prints "No sandboxes found." when empty', async () => {
    const logs = [];
    mock.method(console, 'log', (...a) => logs.push(a.join(' ')));

    const { listCommand } = await importList();
    await listCommand();

    assert.ok(logs.some(l => l.includes('No sandboxes found.')));
  });

  test('prints table with sandbox entries', async () => {
    // Write file directly to avoid module cache issues
    const spekkDir = path.join(tmpDir, '.spekk');
    await fs.mkdir(spekkDir, { recursive: true });
    await fs.writeFile(path.join(spekkDir, 'sandboxes.json'), JSON.stringify({
      'my-box': { ip: '10.0.0.1', region: 'nyc1', status: 'active', createdAt: '2026-01-01' }
    }));

    const logs = [];
    mock.method(console, 'log', (...a) => logs.push(a.join(' ')));

    const { listCommand } = await importList();
    await listCommand();

    const output = logs.join('\n');
    assert.ok(output.includes('my-box'));
    assert.ok(output.includes('10.0.0.1'));
    assert.ok(output.includes('nyc1'));
    assert.ok(output.includes('active'));
  });
});

describe('sandbox status', () => {
  test('exits with error if sandbox not found', async () => {
    let exitCode = null;
    let errorMsg = '';
    mock.method(process, 'exit', (code) => { exitCode = code; throw new Error('exit'); });
    mock.method(console, 'error', (msg) => { errorMsg = msg; });

    const q = `?t=${Date.now()}-${Math.random()}`;
    const { statusCommand } = await import(`../status.js${q}`);

    await assert.rejects(() => statusCommand(['nonexistent']), { message: 'exit' });
    assert.strictEqual(exitCode, 1);
    assert.ok(errorMsg.includes('not found'));
  });

  test('exits with error if no name provided', async () => {
    let exitCode = null;
    mock.method(process, 'exit', (code) => { exitCode = code; throw new Error('exit'); });
    mock.method(console, 'error', () => {});

    const q = `?t=${Date.now()}-${Math.random()}`;
    const { statusCommand } = await import(`../status.js${q}`);

    await assert.rejects(() => statusCommand([]), { message: 'exit' });
    assert.strictEqual(exitCode, 1);
  });
});
