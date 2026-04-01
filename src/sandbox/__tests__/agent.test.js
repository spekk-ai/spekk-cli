import { test, describe, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert';
import childProcess from 'child_process';
import { EventEmitter } from 'events';
import { Readable } from 'stream';

let originalSpawn;

beforeEach(() => {
  originalSpawn = childProcess.spawn;
});

afterEach(() => {
  childProcess.spawn = originalSpawn;
});

// Helper to mock spawn with controlled exit codes
function mockSpawnSuccess(spawnCalls = []) {
  childProcess.spawn = (cmd, args, opts) => {
    spawnCalls.push({ cmd, args });
    const child = new EventEmitter();
    child.stdout = Readable.from(['']);
    child.stderr = Readable.from(['']);
    process.nextTick(() => child.emit('close', 0));
    return child;
  };
}

describe('deployAgent', () => {
  test('rsyncs binary, chmod +x, daemon-reload and restarts spekk-agent, prints version', async () => {
    // Mock fetchReleaseArtifacts by patching the module's dependency
    // We import agent.js fresh and intercept spawn calls
    const spawnCalls = [];
    mockSpawnSuccess(spawnCalls);

    // Patch fetchReleaseArtifacts by using module mock approach
    // Since ESM doesn't allow easy mocking, we test via integration with mocked fetch
    const originalFetch = globalThis.fetch;
    const FAKE_BINARY = Buffer.from([0x7f, 0x45, 0x4c, 0x46]);

    globalThis.fetch = async (url) => {
      if (url.includes('/releases/latest') || url.includes('/releases/tags/')) {
        return {
          ok: true, status: 200, json: async () => ({
            tag_name: 'v2.0.0',
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
      throw new Error(`Unexpected URL: ${url}`);
    };

    process.env.GITHUB_TOKEN = 'test-token';

    const logs = [];
    const origLog = console.log;
    console.log = (...args) => logs.push(args.join(' '));

    try {
      const { deployAgent } = await import(`../agent.js?v=${Date.now()}`);
      await deployAgent('10.0.0.1');
    } finally {
      console.log = origLog;
      globalThis.fetch = originalFetch;
    }

    // rsync was called with the correct destination
    const rsyncCall = spawnCalls.find(c => c.cmd === 'rsync');
    assert.ok(rsyncCall, 'Expected rsync to be called');
    assert.ok(
      rsyncCall.args.some(a => a.includes('10.0.0.1') && a.includes('/opt/spekk/agent-client')),
      `Expected rsync destination to include IP and remote path, got: ${rsyncCall.args}`
    );

    // ssh was called for chmod +x
    const chmodCall = spawnCalls.find(c => c.cmd === 'ssh' && c.args.some(a => a.includes('chmod')));
    assert.ok(chmodCall, 'Expected ssh chmod call');
    assert.ok(chmodCall.args.some(a => a.includes('10.0.0.1')));

    // ssh was called for systemctl
    const systemctlCall = spawnCalls.find(c => c.cmd === 'ssh' && c.args.some(a => a.includes('systemctl')));
    assert.ok(systemctlCall, 'Expected ssh systemctl call');
    assert.ok(systemctlCall.args.some(a => a.includes('daemon-reload')));
    assert.ok(systemctlCall.args.some(a => a.includes('restart spekk-agent')));

    // version was printed
    const allLogs = logs.join(' ');
    assert.ok(allLogs.includes('v2.0.0'), `Expected version in output, got: ${allLogs}`);
  });

  test('deployAgent module exports deployAgent function', async () => {
    const { deployAgent } = await import('../agent.js');
    assert.strictEqual(typeof deployAgent, 'function');
  });
});
