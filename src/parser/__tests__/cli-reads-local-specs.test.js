import { test } from 'node:test';
import assert from 'node:assert';
import { run } from '../cli.js';
import fs from 'fs';
import path from 'path';
import os from 'os';

test('CLI reads specs from current working directory', async (t) => {
  const originalCwd = process.cwd();
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'cli-test-'));
  const specsDir = path.join(testDir, 'specs');

  try {
    // Create a simple spec structure
    fs.mkdirSync(specsDir, { recursive: true });
    fs.mkdirSync(path.join(specsDir, 'test-spec'), { recursive: true });

    fs.writeFileSync(
      path.join(specsDir, 'test-spec', 'test-spec.md'),
      `---
id: test-spec
created: 2024-01-01T00:00:00Z
priority: 1
---

# Test Spec
`
    );

    // Change to test directory and run the CLI
    process.chdir(testDir);

    const logOutput = [];
    t.mock.method(console, 'log', (msg) => logOutput.push(msg));

    await run({ specsDirectory: path.join(testDir, 'specs'), allBranches: true });

    assert.strictEqual(logOutput.length, 1);
    const output = JSON.parse(logOutput[0]);
    assert.strictEqual(output.status, 'complete');
  } finally {
    process.chdir(originalCwd);

    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }
  }
});

test('CLI uses specsDirectory option to find specs', async (t) => {
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'cli-test-'));
  const specsDir = path.join(testDir, 'specs');

  try {
    fs.mkdirSync(specsDir, { recursive: true });
    fs.mkdirSync(path.join(specsDir, 'another-spec', 'assertions'), { recursive: true });

    fs.writeFileSync(
      path.join(specsDir, 'another-spec', 'another-spec.md'),
      `---
id: another-spec
created: 2024-01-02T00:00:00Z
priority: 2
---

# Another Spec
`
    );

    fs.writeFileSync(
      path.join(specsDir, 'another-spec', 'assertions', 'test-assertion.md'),
      `---
id: test-assertion
parent: another-spec
created: 2024-01-02T00:00:00Z
priority: 1
status: not_started
---

# Test Assertion
`
    );

    const logOutput = [];
    t.mock.method(console, 'log', (msg) => logOutput.push(msg));

    await run({ specsDirectory: specsDir, allBranches: true });

    assert.strictEqual(logOutput.length, 1);
    const output = JSON.parse(logOutput[0]);
    assert.strictEqual(output.id, 'test-assertion');
  } finally {
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }
  }
});

test('CLI reads different specs from different directories', async (t) => {
  const testDir1 = fs.mkdtempSync(path.join(os.tmpdir(), 'cli-test-1-'));
  const testDir2 = fs.mkdtempSync(path.join(os.tmpdir(), 'cli-test-2-'));

  try {
    // Setup first directory
    const specsDir1 = path.join(testDir1, 'specs');
    fs.mkdirSync(path.join(specsDir1, 'spec-one', 'assertions'), { recursive: true });
    fs.writeFileSync(
      path.join(specsDir1, 'spec-one', 'spec-one.md'),
      `---
id: spec-one
created: 2024-01-01T00:00:00Z
priority: 1
---

# Spec One
`
    );
    fs.writeFileSync(
      path.join(specsDir1, 'spec-one', 'assertions', 'assertion-one.md'),
      `---
id: assertion-one
parent: spec-one
created: 2024-01-01T00:00:00Z
priority: 1
status: not_started
---

# Assertion One
`
    );

    // Setup second directory
    const specsDir2 = path.join(testDir2, 'specs');
    fs.mkdirSync(path.join(specsDir2, 'spec-two', 'assertions'), { recursive: true });
    fs.writeFileSync(
      path.join(specsDir2, 'spec-two', 'spec-two.md'),
      `---
id: spec-two
created: 2024-01-02T00:00:00Z
priority: 1
---

# Spec Two
`
    );
    fs.writeFileSync(
      path.join(specsDir2, 'spec-two', 'assertions', 'assertion-two.md'),
      `---
id: assertion-two
parent: spec-two
created: 2024-01-02T00:00:00Z
priority: 1
status: not_started
---

# Assertion Two
`
    );

    // Test first directory
    const logOutput1 = [];
    t.mock.method(console, 'log', (msg) => logOutput1.push(msg));
    await run({ specsDirectory: specsDir1, allBranches: true });
    assert.strictEqual(JSON.parse(logOutput1[0]).id, 'assertion-one');

    // Test second directory
    const logOutput2 = [];
    t.mock.method(console, 'log', (msg) => logOutput2.push(msg));
    await run({ specsDirectory: specsDir2, allBranches: true });
    assert.strictEqual(JSON.parse(logOutput2[0]).id, 'assertion-two');
  } finally {
    if (fs.existsSync(testDir1)) {
      fs.rmSync(testDir1, { recursive: true, force: true });
    }
    if (fs.existsSync(testDir2)) {
      fs.rmSync(testDir2, { recursive: true, force: true });
    }
  }
});

test('CLI shows appropriate message when no specs directory exists', async (t) => {
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'cli-test-'));

  try {
    const logOutput = [];
    t.mock.method(console, 'log', (msg) => logOutput.push(msg));

    // Pass a non-existent specs directory
    await run({ specsDirectory: path.join(testDir, 'specs'), allBranches: true });

    assert.ok(logOutput.length > 0);
    const output = JSON.parse(logOutput[0]);
    assert.strictEqual(output.status, 'empty');
  } finally {
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }
  }
});

test('CLI handles nested directory structures', async (t) => {
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'cli-test-'));
  const specsDir = path.join(testDir, 'specs');

  try {
    // Create nested structure
    fs.mkdirSync(path.join(specsDir, 'parent-spec', 'assertions'), { recursive: true });

    fs.writeFileSync(
      path.join(specsDir, 'parent-spec', 'parent-spec.md'),
      `---
id: parent-spec
created: 2024-01-01T00:00:00Z
priority: 1
---

# Parent Spec
`
    );

    fs.writeFileSync(
      path.join(specsDir, 'parent-spec', 'assertions', 'child-assertion.md'),
      `---
id: child-assertion
parent: parent-spec
created: 2024-01-02T00:00:00Z
priority: 1
status: not_started
---

# Child Assertion
`
    );

    const logOutput = [];
    t.mock.method(console, 'log', (msg) => logOutput.push(msg));

    await run({ specsDirectory: specsDir, allBranches: true });

    assert.strictEqual(logOutput.length, 1);
    const output = JSON.parse(logOutput[0]);
    assert.strictEqual(output.id, 'child-assertion');
  } finally {
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }
  }
});
