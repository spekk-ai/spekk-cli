import { describe, it, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert';
import { launchSandbox } from '../cli.js';

describe('sandbox CLI routing', () => {
  let originalLog, originalError, originalExitCode;
  let logs, errors;

  beforeEach(() => {
    originalLog = console.log;
    originalError = console.error;
    originalExitCode = process.exitCode;
    logs = [];
    errors = [];
    console.log = (...args) => logs.push(args.join(' '));
    console.error = (...args) => errors.push(args.join(' '));
    process.exitCode = undefined;
  });

  afterEach(() => {
    console.log = originalLog;
    console.error = originalError;
    process.exitCode = originalExitCode;
  });

  it('prints help with all six subcommands when called with --help', async () => {
    await launchSandbox(['--help']);
    const output = logs.join('\n');
    for (const cmd of ['create', 'list', 'status', 'ssh', 'destroy', 'deploy']) {
      assert.ok(output.includes(cmd), `Help should mention "${cmd}"`);
    }
    assert.strictEqual(process.exitCode, 0);
  });

  it('prints help when called with no subcommand', async () => {
    await launchSandbox([]);
    const output = logs.join('\n');
    assert.ok(output.includes('sandbox'), 'Should print sandbox help');
    assert.strictEqual(process.exitCode, 0);
  });

  it('exits with code 1 for unknown subcommand', async () => {
    await launchSandbox(['unknown-cmd']);
    assert.strictEqual(process.exitCode, 1);
    assert.ok(errors.some(e => e.includes('Unknown sandbox command')));
  });

  it('create subcommand requires --name flag', async () => {
    await launchSandbox(['create']);
    assert.strictEqual(process.exitCode, 1);
    assert.ok(errors.some(e => e.includes('--name is required')));
  });
});
