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

    // Mock spawn so scp/ssh are never actually executed
    const spawnCalls = [];
    childProcess.spawn = (cmd, args, opts) => {
      spawnCalls.push({ cmd, args });
      const child = new EventEmitter();
      const stdoutData = (cmd === 'ssh' && args.includes('systemctl is-active spekk-agent')) ? 'active\n' : '';
      child.stdout = Readable.from([stdoutData]);
      child.stderr = Readable.from([]);
      process.nextTick(() => child.emit('close', 0));
      return child;
    };

    const originalLog = console.log;
    let logMsgs = [];
    console.log = (msg) => { logMsgs.push(msg); };

    try {
      await deployCommand(['test-deploy']);
    } finally {
      console.log = originalLog;
    }

    // Verify the sandbox IP was used in console output
    const allOutput = logMsgs.join(' ');
    assert.ok(
      allOutput.includes('192.168.1.100'),
      `Expected IP 192.168.1.100 in output, got: ${allOutput}`
    );

    // Verify scp was called with the correct IP
    const scpCall = spawnCalls.find(c => c.cmd === 'scp');
    assert.ok(scpCall, 'Expected scp to be called');
    assert.ok(
      scpCall.args.some(a => a.includes('192.168.1.100')),
      `Expected scp args to include IP, got: ${scpCall.args}`
    );
  });
});
