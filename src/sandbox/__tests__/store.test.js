import { test, describe, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert';
import fs from 'fs/promises';
import path from 'path';
import os from 'os';

// Override HOME to isolate tests from real ~/.spekk
let originalHome;
let tmpDir;

beforeEach(async () => {
  tmpDir = await fs.mkdtemp(path.join(os.tmpdir(), 'spekk-store-test-'));
  originalHome = process.env.HOME;
  process.env.HOME = tmpDir;
});

afterEach(async () => {
  process.env.HOME = originalHome;
  await fs.rm(tmpDir, { recursive: true, force: true });
});

// Dynamic import to pick up the overridden HOME each time
async function importStore() {
  // We need to bust the module cache to pick up the new HOME
  const uniqueQuery = `?t=${Date.now()}-${Math.random()}`;
  return import(`../store.js${uniqueQuery}`);
}

describe('sandbox metadata store', () => {
  test('loadSandboxes returns empty object when file does not exist', async () => {
    const { loadSandboxes } = await importStore();
    const result = await loadSandboxes();
    assert.deepStrictEqual(result, {});
  });

  test('saveSandbox creates ~/.spekk directory and file', async () => {
    const { saveSandbox, loadSandboxes } = await importStore();
    await saveSandbox('test-box', { dropletId: 123, ip: '1.2.3.4', region: 'nyc1', size: 's-1vcpu-1gb', createdAt: '2026-01-01', status: 'active' });
    const all = await loadSandboxes();
    assert.strictEqual(all['test-box'].dropletId, 123);
    assert.strictEqual(all['test-box'].ip, '1.2.3.4');
  });

  test('saveSandbox merges data into existing entry', async () => {
    const { saveSandbox, getSandbox } = await importStore();
    await saveSandbox('box1', { dropletId: 1, ip: '10.0.0.1' });
    await saveSandbox('box1', { status: 'active' });
    const entry = await getSandbox('box1');
    assert.strictEqual(entry.dropletId, 1);
    assert.strictEqual(entry.ip, '10.0.0.1');
    assert.strictEqual(entry.status, 'active');
  });

  test('getSandbox returns null for non-existent name', async () => {
    const { getSandbox } = await importStore();
    const result = await getSandbox('no-such-box');
    assert.strictEqual(result, null);
  });

  test('removeSandbox deletes the entry', async () => {
    const { saveSandbox, removeSandbox, getSandbox, loadSandboxes } = await importStore();
    await saveSandbox('a', { dropletId: 1 });
    await saveSandbox('b', { dropletId: 2 });
    await removeSandbox('a');
    assert.strictEqual(await getSandbox('a'), null);
    assert.strictEqual((await loadSandboxes())['b'].dropletId, 2);
  });
});
