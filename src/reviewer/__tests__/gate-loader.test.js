import { describe, test, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert';
import fs from 'fs';
import path from 'path';
import os from 'os';
import { loadGates, getGatePaths } from '../gate-loader.js';

function createTempDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-gate-test-'));
}

function cleanup(dir) {
  if (fs.existsSync(dir)) fs.rmSync(dir, { recursive: true, force: true });
}

const SAMPLE_GATE = `---
id: validate-testids
phase: post-build
tags:
  - frontend
  - testing
depends-on: api-audit
---

# Validate Test IDs

## Preconditions
- files-changed: "**/*.{tsx,jsx}"
- dir-exists: "src/components"

## LLM Judgment
Skip if the only JSX changes are in test files or storybook files.

## Workflow
Run validate-testids --fix on all changed components.

## On Failure
- severity: warning
- action: report
`;

const MINIMAL_GATE = `---
id: simple-check
phase: pre-merge
---

# Simple Check

## Preconditions
- file-exists: "package.json"

## Workflow
Check that package.json is valid JSON.
`;

describe('Gate Loader', () => {
  let tempDir;

  beforeEach(() => {
    tempDir = createTempDir();
  });

  afterEach(() => {
    cleanup(tempDir);
  });

  test('loads gate files from a single directory', () => {
    const gatesDir = path.join(tempDir, 'gates');
    fs.mkdirSync(gatesDir, { recursive: true });
    fs.writeFileSync(path.join(gatesDir, 'validate-testids.gate.md'), SAMPLE_GATE);

    const gates = loadGates({ packageRoot: tempDir, globalDir: '/nonexistent', localDir: '/nonexistent' });

    assert.equal(gates.length, 1);
    assert.equal(gates[0].id, 'validate-testids');
    assert.equal(gates[0].phase, 'post-build');
    assert.deepStrictEqual(gates[0].tags, ['frontend', 'testing']);
    assert.equal(gates[0].dependsOn, 'api-audit');
    assert.equal(gates[0].layer, 'package');
    assert.equal(gates[0].title, 'Validate Test IDs');
  });

  test('parses preconditions into structured check objects', () => {
    const gatesDir = path.join(tempDir, 'gates');
    fs.mkdirSync(gatesDir, { recursive: true });
    fs.writeFileSync(path.join(gatesDir, 'test.gate.md'), `---
id: all-checks
phase: post-build
---

# All Check Types

## Preconditions
- files-changed: "**/*.tsx"
- dir-exists: "src/components"
- file-exists: "playwright.config.ts"
- file-not-exists: ".skip-review"
- branch-matches: "feature/*"
- has-dependency: "@playwright/test"
- command-succeeds: "gh pr view"
`);

    const gates = loadGates({ packageRoot: tempDir, globalDir: '/nonexistent', localDir: '/nonexistent' });

    assert.equal(gates[0].preconditions.length, 7);
    assert.deepStrictEqual(gates[0].preconditions[0], { type: 'files-changed', pattern: '**/*.tsx' });
    assert.deepStrictEqual(gates[0].preconditions[1], { type: 'dir-exists', path: 'src/components' });
    assert.deepStrictEqual(gates[0].preconditions[2], { type: 'file-exists', path: 'playwright.config.ts' });
    assert.deepStrictEqual(gates[0].preconditions[3], { type: 'file-not-exists', path: '.skip-review' });
    assert.deepStrictEqual(gates[0].preconditions[4], { type: 'branch-matches', pattern: 'feature/*' });
    assert.deepStrictEqual(gates[0].preconditions[5], { type: 'has-dependency', package: '@playwright/test' });
    assert.deepStrictEqual(gates[0].preconditions[6], { type: 'command-succeeds', command: 'gh pr view' });
  });

  test('parses LLM Judgment section as raw text', () => {
    const gatesDir = path.join(tempDir, 'gates');
    fs.mkdirSync(gatesDir, { recursive: true });
    fs.writeFileSync(path.join(gatesDir, 'test.gate.md'), SAMPLE_GATE);

    const gates = loadGates({ packageRoot: tempDir, globalDir: '/nonexistent', localDir: '/nonexistent' });

    assert.equal(gates[0].llmJudgment, 'Skip if the only JSX changes are in test files or storybook files.');
  });

  test('parses Workflow section as raw text', () => {
    const gatesDir = path.join(tempDir, 'gates');
    fs.mkdirSync(gatesDir, { recursive: true });
    fs.writeFileSync(path.join(gatesDir, 'test.gate.md'), SAMPLE_GATE);

    const gates = loadGates({ packageRoot: tempDir, globalDir: '/nonexistent', localDir: '/nonexistent' });

    assert.equal(gates[0].workflow, 'Run validate-testids --fix on all changed components.');
  });

  test('parses On Failure section for severity and action', () => {
    const gatesDir = path.join(tempDir, 'gates');
    fs.mkdirSync(gatesDir, { recursive: true });
    fs.writeFileSync(path.join(gatesDir, 'test.gate.md'), SAMPLE_GATE);

    const gates = loadGates({ packageRoot: tempDir, globalDir: '/nonexistent', localDir: '/nonexistent' });

    assert.deepStrictEqual(gates[0].onFailure, { severity: 'warning', action: 'report' });
  });

  test('local gates override package gates by id', () => {
    const packageGatesDir = path.join(tempDir, 'pkg', 'gates');
    const localGatesDir = path.join(tempDir, 'local', 'gates');
    fs.mkdirSync(packageGatesDir, { recursive: true });
    fs.mkdirSync(localGatesDir, { recursive: true });

    fs.writeFileSync(path.join(packageGatesDir, 'check.gate.md'), `---
id: my-check
phase: post-build
---

# Package Check

## Workflow
Package version of the check.
`);

    fs.writeFileSync(path.join(localGatesDir, 'check.gate.md'), `---
id: my-check
phase: pre-merge
---

# Local Check

## Workflow
Local override of the check.
`);

    const gates = loadGates({
      packageRoot: path.join(tempDir, 'pkg'),
      globalDir: '/nonexistent',
      localDir: localGatesDir,
    });

    assert.equal(gates.length, 1);
    assert.equal(gates[0].id, 'my-check');
    assert.equal(gates[0].phase, 'pre-merge');
    assert.equal(gates[0].layer, 'local');
    assert.equal(gates[0].workflow, 'Local override of the check.');
  });

  test('global gates override package gates by id', () => {
    const packageGatesDir = path.join(tempDir, 'pkg', 'gates');
    const globalGatesDir = path.join(tempDir, 'global', 'gates');
    fs.mkdirSync(packageGatesDir, { recursive: true });
    fs.mkdirSync(globalGatesDir, { recursive: true });

    fs.writeFileSync(path.join(packageGatesDir, 'check.gate.md'), `---
id: my-check
phase: post-build
---

# Package Check

## Workflow
Package version.
`);

    fs.writeFileSync(path.join(globalGatesDir, 'check.gate.md'), `---
id: my-check
phase: post-build
---

# Global Check

## Workflow
Global version.
`);

    const gates = loadGates({
      packageRoot: path.join(tempDir, 'pkg'),
      globalDir: globalGatesDir,
      localDir: '/nonexistent',
    });

    assert.equal(gates.length, 1);
    assert.equal(gates[0].layer, 'global');
    assert.equal(gates[0].workflow, 'Global version.');
  });

  test('returns gates sorted by dependency order', () => {
    const gatesDir = path.join(tempDir, 'gates');
    fs.mkdirSync(gatesDir, { recursive: true });

    fs.writeFileSync(path.join(gatesDir, 'b-downstream.gate.md'), `---
id: downstream
phase: post-build
depends-on: upstream
---

# Downstream

## Workflow
Runs after upstream.
`);

    fs.writeFileSync(path.join(gatesDir, 'a-upstream.gate.md'), `---
id: upstream
phase: post-build
---

# Upstream

## Workflow
Runs first.
`);

    const gates = loadGates({ packageRoot: tempDir, globalDir: '/nonexistent', localDir: '/nonexistent' });

    assert.equal(gates.length, 2);
    assert.equal(gates[0].id, 'upstream');
    assert.equal(gates[1].id, 'downstream');
  });

  test('handles missing optional sections gracefully', () => {
    const gatesDir = path.join(tempDir, 'gates');
    fs.mkdirSync(gatesDir, { recursive: true });
    fs.writeFileSync(path.join(gatesDir, 'minimal.gate.md'), MINIMAL_GATE);

    const gates = loadGates({ packageRoot: tempDir, globalDir: '/nonexistent', localDir: '/nonexistent' });

    assert.equal(gates[0].id, 'simple-check');
    assert.equal(gates[0].llmJudgment, null);
    assert.equal(gates[0].onFailure.severity, undefined);
    assert.deepStrictEqual(gates[0].tags, []);
    assert.equal(gates[0].dependsOn, null);
  });

  test('skips non-.gate.md files', () => {
    const gatesDir = path.join(tempDir, 'gates');
    fs.mkdirSync(gatesDir, { recursive: true });
    fs.writeFileSync(path.join(gatesDir, 'valid.gate.md'), MINIMAL_GATE);
    fs.writeFileSync(path.join(gatesDir, 'readme.md'), '# Not a gate');
    fs.writeFileSync(path.join(gatesDir, 'notes.txt'), 'just notes');

    const gates = loadGates({ packageRoot: tempDir, globalDir: '/nonexistent', localDir: '/nonexistent' });

    assert.equal(gates.length, 1);
    assert.equal(gates[0].id, 'simple-check');
  });

  test('returns empty array when no gate directories exist', () => {
    const gates = loadGates({ packageRoot: '/nonexistent', globalDir: '/nonexistent', localDir: '/nonexistent' });
    assert.deepStrictEqual(gates, []);
  });

  test('skips files without frontmatter', () => {
    const gatesDir = path.join(tempDir, 'gates');
    fs.mkdirSync(gatesDir, { recursive: true });
    fs.writeFileSync(path.join(gatesDir, 'nofm.gate.md'), '# No frontmatter here');

    const gates = loadGates({ packageRoot: tempDir, globalDir: '/nonexistent', localDir: '/nonexistent' });
    assert.deepStrictEqual(gates, []);
  });

  test('three-layer resolution with all layers present', () => {
    const pkgDir = path.join(tempDir, 'pkg', 'gates');
    const globalDir = path.join(tempDir, 'global');
    const localDir = path.join(tempDir, 'local');
    fs.mkdirSync(pkgDir, { recursive: true });
    fs.mkdirSync(globalDir, { recursive: true });
    fs.mkdirSync(localDir, { recursive: true });

    // Package gate (will be kept — unique id)
    fs.writeFileSync(path.join(pkgDir, 'pkg-only.gate.md'), `---
id: pkg-only
phase: post-build
---

# Package Only

## Workflow
Only in package.
`);

    // Global gate (will be kept — unique id)
    fs.writeFileSync(path.join(globalDir, 'global-only.gate.md'), `---
id: global-only
phase: post-build
---

# Global Only

## Workflow
Only in global.
`);

    // Shared gate — exists at all three layers, local wins
    fs.writeFileSync(path.join(pkgDir, 'shared.gate.md'), `---
id: shared
phase: post-build
---

# Shared Package

## Workflow
Package version.
`);

    fs.writeFileSync(path.join(globalDir, 'shared.gate.md'), `---
id: shared
phase: post-build
---

# Shared Global

## Workflow
Global version.
`);

    fs.writeFileSync(path.join(localDir, 'shared.gate.md'), `---
id: shared
phase: pre-merge
---

# Shared Local

## Workflow
Local version.
`);

    const gates = loadGates({
      packageRoot: path.join(tempDir, 'pkg'),
      globalDir,
      localDir,
    });

    assert.equal(gates.length, 3);
    const shared = gates.find(g => g.id === 'shared');
    assert.equal(shared.layer, 'local');
    assert.equal(shared.phase, 'pre-merge');
    assert.ok(gates.find(g => g.id === 'pkg-only'));
    assert.ok(gates.find(g => g.id === 'global-only'));
  });
});

describe('getGatePaths', () => {
  test('returns three paths in package → global → local order', () => {
    const paths = getGatePaths({ packageRoot: '/pkg', globalDir: '/global', localDir: '/local' });
    assert.equal(paths.length, 3);
    assert.equal(paths[0].layer, 'package');
    assert.equal(paths[1].layer, 'global');
    assert.equal(paths[2].layer, 'local');
    assert.ok(paths[0].path.endsWith('gates'));
    assert.equal(paths[1].path, '/global');
    assert.equal(paths[2].path, '/local');
  });
});
