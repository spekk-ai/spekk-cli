import { test, describe, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert';
import fs from 'fs/promises';
import path from 'path';
import os from 'os';
import childProcess from 'child_process';
import { EventEmitter } from 'events';
import { Readable } from 'stream';
import { deployCommand } from '../deploy.js';

let originalHome;
let originalSpawn;
let tmpDir;

beforeEach(async () => {
  tmpDir = await fs.mkdtemp(path.join(os.tmpdir(), 'spekk-deploy-test-'));
  originalHome = process.env.HOME;
  originalSpawn = childProcess.spawn;
  process.env.HOME = tmpDir;
});

afterEach(async () => {
  process.env.HOME = originalHome;
  childProcess.spawn = originalSpawn;
  await fs.rm(tmpDir, { recursive: true, force: true });
});

async function writeSandboxFile(sandboxes) {
  const spekkDir = path.join(tmpDir, '.spekk');
  await fs.mkdir(spekkDir, { recursive: true });
  await fs.writeFile(path.join(spekkDir, 'sandboxes.json'), JSON.stringify(sandboxes, null, 2));
}

describe('sandbox deploy', () => {
  test('exits with code 1 when sandbox not found', async () => {
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

  test('looks up sandbox from store and deploys to correct IP', async () => {
    await writeSandboxFile({
      'test-deploy': { dropletId: 999, ip: '192.168.1.100', region: 'nyc1', status: 'active' }
    });

    const spawnCalls = [];
    childProcess.spawn = (cmd, args, opts) => {
      spawnCalls.push({ cmd, args });
      const child = new EventEmitter();
      child.stdout = Readable.from(['']);
      child.stderr = Readable.from([]);
      process.nextTick(() => child.emit('close', 0));
      return child;
    };

    const originalFetch = globalThis.fetch;
    const FAKE_BINARY = Buffer.from([0x7f, 0x45, 0x4c, 0x46]);
    globalThis.fetch = async (url) => {
      if (url.includes('/releases/latest') || url.includes('/releases/tags/')) {
        return {
          ok: true, status: 200, json: async () => ({
            tag_name: 'v1.0.0',
            assets: [
              { name: 'sandbox', id: 1001 },
              { name: 'cloud-init.yaml', id: 1002 },
            ],
          }),
        };
      }
      if (url.includes('/releases/assets/')) {
        return { ok: true, status: 200, arrayBuffer: async () => FAKE_BINARY.buffer, text: async () => '#cloud-config\n' };
      }
    };

    process.env.GITHUB_TOKEN = 'test-token';

    const originalLog = console.log;
    let logMsgs = [];
    console.log = (msg) => { logMsgs.push(msg); };

    try {
      await deployCommand(['test-deploy']);
    } finally {
      console.log = originalLog;
      globalThis.fetch = originalFetch;
    }

    const rsyncCall = spawnCalls.find(c => c.cmd === 'rsync');
    assert.ok(rsyncCall, 'Expected rsync to be called');
    assert.ok(
      rsyncCall.args.some(a => a.includes('192.168.1.100')),
      `Expected rsync args to include IP, got: ${rsyncCall.args}`
    );
  });
});
