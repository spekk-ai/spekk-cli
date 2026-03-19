import { describe, test, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert';
import fs from 'fs';
import path from 'path';
import os from 'os';
import { evaluateGates } from '../gate-engine.js';

function createTempDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-engine-test-'));
}

function cleanup(dir) {
  if (fs.existsSync(dir)) fs.rmSync(dir, { recursive: true, force: true });
}

function makeGate(overrides = {}) {
  return {
    id: 'test-gate',
    phase: 'post-build',
    tags: [],
    dependsOn: null,
    title: 'Test Gate',
    layer: 'package',
    file: '/fake/path.gate.md',
    preconditions: [],
    llmJudgment: null,
    workflow: null,
    onFailure: {},
    ...overrides,
  };
}

describe('Gate Engine', () => {
  let tempDir;

  beforeEach(() => {
    tempDir = createTempDir();
  });

  afterEach(() => {
    cleanup(tempDir);
  });

  test('gate with no preconditions passes', () => {
    const gates = [makeGate()];
    const results = evaluateGates(gates, { cwd: tempDir, branch: 'main', changedFiles: [], dependencies: {} });

    assert.equal(results.length, 1);
    assert.equal(results[0].id, 'test-gate');
    assert.equal(results[0].status, 'pass');
    assert.equal(results[0].reason, null);
  });

  test('dir-exists passes when directory exists', () => {
    fs.mkdirSync(path.join(tempDir, 'src', 'components'), { recursive: true });

    const gates = [makeGate({
      preconditions: [{ type: 'dir-exists', path: 'src/components' }],
    })];

    const results = evaluateGates(gates, { cwd: tempDir, branch: 'main', changedFiles: [], dependencies: {} });
    assert.equal(results[0].status, 'pass');
  });

  test('dir-exists skips when directory missing', () => {
    const gates = [makeGate({
      preconditions: [{ type: 'dir-exists', path: 'src/components' }],
    })];

    const results = evaluateGates(gates, { cwd: tempDir, branch: 'main', changedFiles: [], dependencies: {} });
    assert.equal(results[0].status, 'skip');
    assert.equal(results[0].reason, 'directory not found: src/components');
  });

  test('file-exists passes when file exists', () => {
    fs.writeFileSync(path.join(tempDir, 'package.json'), '{}');

    const gates = [makeGate({
      preconditions: [{ type: 'file-exists', path: 'package.json' }],
    })];

    const results = evaluateGates(gates, { cwd: tempDir, branch: 'main', changedFiles: [], dependencies: {} });
    assert.equal(results[0].status, 'pass');
  });

  test('file-exists skips when file missing', () => {
    const gates = [makeGate({
      preconditions: [{ type: 'file-exists', path: 'missing.json' }],
    })];

    const results = evaluateGates(gates, { cwd: tempDir, branch: 'main', changedFiles: [], dependencies: {} });
    assert.equal(results[0].status, 'skip');
    assert.equal(results[0].reason, 'file not found: missing.json');
  });

  test('file-not-exists passes when file is absent', () => {
    const gates = [makeGate({
      preconditions: [{ type: 'file-not-exists', path: '.skip-review' }],
    })];

    const results = evaluateGates(gates, { cwd: tempDir, branch: 'main', changedFiles: [], dependencies: {} });
    assert.equal(results[0].status, 'pass');
  });

  test('file-not-exists skips when file present', () => {
    fs.writeFileSync(path.join(tempDir, '.skip-review'), '');

    const gates = [makeGate({
      preconditions: [{ type: 'file-not-exists', path: '.skip-review' }],
    })];

    const results = evaluateGates(gates, { cwd: tempDir, branch: 'main', changedFiles: [], dependencies: {} });
    assert.equal(results[0].status, 'skip');
    assert.equal(results[0].reason, 'file exists: .skip-review');
  });

  test('has-dependency passes when dependency present', () => {
    const gates = [makeGate({
      preconditions: [{ type: 'has-dependency', package: '@playwright/test' }],
    })];

    const results = evaluateGates(gates, {
      cwd: tempDir, branch: 'main', changedFiles: [],
      dependencies: { '@playwright/test': '^1.0.0' },
    });
    assert.equal(results[0].status, 'pass');
  });

  test('has-dependency skips when dependency missing', () => {
    const gates = [makeGate({
      preconditions: [{ type: 'has-dependency', package: '@playwright/test' }],
    })];

    const results = evaluateGates(gates, { cwd: tempDir, branch: 'main', changedFiles: [], dependencies: {} });
    assert.equal(results[0].status, 'skip');
    assert.equal(results[0].reason, 'dependency not found: @playwright/test');
  });

  test('branch-matches passes on matching branch', () => {
    const gates = [makeGate({
      preconditions: [{ type: 'branch-matches', pattern: 'feature/*' }],
    })];

    const results = evaluateGates(gates, { cwd: tempDir, branch: 'feature/my-feature', changedFiles: [], dependencies: {} });
    assert.equal(results[0].status, 'pass');
  });

  test('branch-matches skips on non-matching branch', () => {
    const gates = [makeGate({
      preconditions: [{ type: 'branch-matches', pattern: 'feature/*' }],
    })];

    const results = evaluateGates(gates, { cwd: tempDir, branch: 'main', changedFiles: [], dependencies: {} });
    assert.equal(results[0].status, 'skip');
    assert.ok(results[0].reason.includes('does not match'));
  });

  test('files-changed passes when matching files exist in diff', () => {
    const gates = [makeGate({
      preconditions: [{ type: 'files-changed', pattern: '**/*.tsx' }],
    })];

    const results = evaluateGates(gates, {
      cwd: tempDir, branch: 'feature/x',
      changedFiles: ['src/App.tsx', 'README.md'],
      dependencies: {},
    });
    assert.equal(results[0].status, 'pass');
  });

  test('files-changed skips when no matching files in diff', () => {
    const gates = [makeGate({
      preconditions: [{ type: 'files-changed', pattern: '**/*.tsx' }],
    })];

    const results = evaluateGates(gates, {
      cwd: tempDir, branch: 'feature/x',
      changedFiles: ['README.md', 'package.json'],
      dependencies: {},
    });
    assert.equal(results[0].status, 'skip');
    assert.equal(results[0].reason, 'no **/*.tsx files changed on branch');
  });

  test('command-succeeds passes when command exits 0', () => {
    const gates = [makeGate({
      preconditions: [{ type: 'command-succeeds', command: 'true' }],
    })];

    const results = evaluateGates(gates, { cwd: tempDir, branch: 'main', changedFiles: [], dependencies: {} });
    assert.equal(results[0].status, 'pass');
  });

  test('command-succeeds skips when command fails', () => {
    const gates = [makeGate({
      preconditions: [{ type: 'command-succeeds', command: 'false' }],
    })];

    const results = evaluateGates(gates, { cwd: tempDir, branch: 'main', changedFiles: [], dependencies: {} });
    assert.equal(results[0].status, 'skip');
    assert.equal(results[0].reason, 'command failed: false');
  });

  test('ALL preconditions must pass (AND logic)', () => {
    fs.mkdirSync(path.join(tempDir, 'src', 'components'), { recursive: true });

    const gates = [makeGate({
      preconditions: [
        { type: 'dir-exists', path: 'src/components' },
        { type: 'files-changed', pattern: '**/*.tsx' },
      ],
    })];

    // dir exists but no tsx files changed → skip
    const results = evaluateGates(gates, {
      cwd: tempDir, branch: 'main',
      changedFiles: ['README.md'],
      dependencies: {},
    });
    assert.equal(results[0].status, 'skip');
    assert.equal(results[0].reason, 'no **/*.tsx files changed on branch');
  });

  test('both preconditions pass → gate passes', () => {
    fs.mkdirSync(path.join(tempDir, 'src', 'components'), { recursive: true });

    const gates = [makeGate({
      preconditions: [
        { type: 'dir-exists', path: 'src/components' },
        { type: 'files-changed', pattern: '**/*.tsx' },
      ],
    })];

    const results = evaluateGates(gates, {
      cwd: tempDir, branch: 'main',
      changedFiles: ['src/components/Button.tsx'],
      dependencies: {},
    });
    assert.equal(results[0].status, 'pass');
  });

  test('DAG dependency: skipped upstream auto-skips downstream', () => {
    const gates = [
      makeGate({ id: 'upstream', preconditions: [{ type: 'file-exists', path: 'nope.txt' }] }),
      makeGate({ id: 'downstream', dependsOn: 'upstream', preconditions: [] }),
    ];

    const results = evaluateGates(gates, { cwd: tempDir, branch: 'main', changedFiles: [], dependencies: {} });

    const upstream = results.find(r => r.id === 'upstream');
    const downstream = results.find(r => r.id === 'downstream');

    assert.equal(upstream.status, 'skip');
    assert.equal(downstream.status, 'skip');
    assert.equal(downstream.reason, 'dependency skipped: upstream');
  });

  test('DAG dependency: passed upstream allows downstream to evaluate', () => {
    fs.writeFileSync(path.join(tempDir, 'exists.txt'), '');

    const gates = [
      makeGate({ id: 'upstream', preconditions: [{ type: 'file-exists', path: 'exists.txt' }] }),
      makeGate({ id: 'downstream', dependsOn: 'upstream', preconditions: [] }),
    ];

    const results = evaluateGates(gates, { cwd: tempDir, branch: 'main', changedFiles: [], dependencies: {} });

    assert.equal(results.find(r => r.id === 'upstream').status, 'pass');
    assert.equal(results.find(r => r.id === 'downstream').status, 'pass');
  });

  test('topological sort ensures correct evaluation order', () => {
    // Present gates in reverse dependency order
    const gates = [
      makeGate({ id: 'c', dependsOn: 'b' }),
      makeGate({ id: 'a' }),
      makeGate({ id: 'b', dependsOn: 'a' }),
    ];

    const results = evaluateGates(gates, { cwd: tempDir, branch: 'main', changedFiles: [], dependencies: {} });

    assert.equal(results[0].id, 'a');
    assert.equal(results[1].id, 'b');
    assert.equal(results[2].id, 'c');
  });

  test('circular dependency throws clear error', () => {
    const gates = [
      makeGate({ id: 'a', dependsOn: 'b' }),
      makeGate({ id: 'b', dependsOn: 'a' }),
    ];

    assert.throws(
      () => evaluateGates(gates, { cwd: tempDir, branch: 'main', changedFiles: [], dependencies: {} }),
      /Circular dependency detected in gates/
    );
  });

  test('specific skip reasons for each check type', () => {
    const gates = [
      makeGate({ id: 'g1', preconditions: [{ type: 'dir-exists', path: 'ios/' }] }),
      makeGate({ id: 'g2', preconditions: [{ type: 'has-dependency', package: 'react-native' }] }),
      makeGate({ id: 'g3', preconditions: [{ type: 'files-changed', pattern: '**/*.swift' }] }),
    ];

    const results = evaluateGates(gates, { cwd: tempDir, branch: 'main', changedFiles: [], dependencies: {} });

    assert.equal(results[0].reason, 'directory not found: ios/');
    assert.equal(results[1].reason, 'dependency not found: react-native');
    assert.equal(results[2].reason, 'no **/*.swift files changed on branch');
  });
});
