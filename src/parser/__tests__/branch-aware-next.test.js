import { test } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'child_process';
import { run } from '../index.js';

describe('Branch-aware spekk next', () => {
  let originalLog;
  let originalWarn;
  let logOutput = [];
  let warnOutput = [];

  beforeEach(() => {
    logOutput = [];
    warnOutput = [];
    originalLog = console.log;
    originalWarn = console.warn;
    console.log = (...args) => logOutput.push(args.join(' '));
    console.warn = (...args) => warnOutput.push(args.join(' '));
  });

  afterEach(() => {
    console.log = originalLog;
    console.warn = originalWarn;
  });

  describe('Branch filtering', () => {
    test('returns assertion matching current git branch', () => {
      // Get actual current branch
      const currentBranch = execSync('git branch --show-current', { 
        encoding: 'utf8',
        stdio: ['pipe', 'pipe', 'ignore']
      }).trim();

      run({ specsDirectory: 'src/parser/__tests__/fixtures/branch-aware' });

      const output = JSON.parse(logOutput[0]);
      
      // Should return an assertion for the current branch or one without branch field
      assert.strictEqual(output.type, 'assertion');
      if (output.branch) {
        assert.strictEqual(output.branch, currentBranch);
      }
    });

    test('--all-branches returns assertions from all branches', () => {
      run({ 
        specsDirectory: 'src/parser/__tests__/fixtures/branch-aware',
        allBranches: true
      });

      const output = JSON.parse(logOutput[0]);
      assert.strictEqual(output.type, 'assertion');
      // Should return highest priority regardless of branch
      assert.strictEqual(output.priority, 1);
    });
  });

  describe('Dependency blocking', () => {
    test('filters out assertions with incomplete dependencies', () => {
      run({ specsDirectory: 'src/parser/__tests__/fixtures/dependencies' });

      const output = JSON.parse(logOutput[0]);
      assert.strictEqual(output.type, 'assertion');
      // Should return websocket-connection (no dependency)
      assert.strictEqual(output.id, 'websocket-connection');
      assert.strictEqual(output.dependsOn, undefined);
    });

    test('returns assertion when dependency is satisfied', () => {
      run({ specsDirectory: 'src/parser/__tests__/fixtures/dependencies-satisfied' });

      const output = JSON.parse(logOutput[0]);
      assert.strictEqual(output.type, 'assertion');
      // Should return chat-session-model (dependency done)
      assert.strictEqual(output.id, 'chat-session-model');
      assert.strictEqual(output.dependsOn, 'websocket-connection');
    });
  });

  describe('Priority and timestamp sorting', () => {
    test('sorts by priority first (lower number = higher priority)', () => {
      run({ specsDirectory: 'src/parser/__tests__/fixtures/priority-sort' });

      const output = JSON.parse(logOutput[0]);
      assert.strictEqual(output.type, 'assertion');
      assert.strictEqual(output.priority, 1);
      assert.strictEqual(output.id, 'priority-one');
    });

    test('breaks priority ties with creation timestamp (older first)', () => {
      run({ specsDirectory: 'src/parser/__tests__/fixtures/timestamp-tiebreak' });

      const output = JSON.parse(logOutput[0]);
      assert.strictEqual(output.type, 'assertion');
      assert.strictEqual(output.id, 'older-assertion');
      assert.strictEqual(output.created, '2026-01-01T00:00:00Z');
    });
  });

  describe('Backwards compatibility', () => {
    test('assertions without branch field are available on all branches', () => {
      run({ specsDirectory: 'src/parser/__tests__/fixtures/branch-aware' });

      const output = JSON.parse(logOutput[0]);
      assert.strictEqual(output.type, 'assertion');
      // Should successfully return an assertion (doesn't error out)
      assert.ok(output.id);
    });
  });
});
