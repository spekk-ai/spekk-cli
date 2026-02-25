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
    
    await run({ specsDirectory: 'specs', allBranches: true });
    
    assert.strictEqual(logOutput.length, 1);
    const output = JSON.parse(logOutput[0]);
    assert.strictEqual(output.id, 'test-spec');
  } finally {
    // CRITICAL: Restore working directory FIRST, before any cleanup
    process.chdir(originalCwd);
    
    // Then cleanup temp directory
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }
  }
});

test('CLI uses process.cwd() for spekk next command', async (t) => {
  const originalCwd = process.cwd();
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'cli-test-'));
  const specsDir = path.join(testDir, 'specs');
  
  try {
    fs.mkdirSync(specsDir, { recursive: true });
    fs.mkdirSync(path.join(specsDir, 'another-spec'), { recursive: true });
    
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
    
    process.chdir(testDir);
    
    const logOutput = [];
    t.mock.method(console, 'log', (msg) => logOutput.push(msg));
    
    await run({ allBranches: true });
    
    assert.strictEqual(logOutput.length, 1);
    const output = JSON.parse(logOutput[0]);
    assert.strictEqual(output.id, 'another-spec');
  } finally {
    // CRITICAL: Restore working directory FIRST
    process.chdir(originalCwd);
    
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }
  }
});

test('CLI reads different specs from different directories', async (t) => {
  const originalCwd = process.cwd();
  const testDir1 = fs.mkdtempSync(path.join(os.tmpdir(), 'cli-test-1-'));
  const testDir2 = fs.mkdtempSync(path.join(os.tmpdir(), 'cli-test-2-'));
  
  try {
    // Setup first directory
    const specsDir1 = path.join(testDir1, 'specs');
    fs.mkdirSync(specsDir1, { recursive: true });
    fs.mkdirSync(path.join(specsDir1, 'spec-one'), { recursive: true });
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
    
    // Setup second directory
    const specsDir2 = path.join(testDir2, 'specs');
    fs.mkdirSync(specsDir2, { recursive: true });
    fs.mkdirSync(path.join(specsDir2, 'spec-two'), { recursive: true });
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
    
    // Test first directory
    process.chdir(testDir1);
    const logOutput1 = [];
    t.mock.method(console, 'log', (msg) => logOutput1.push(msg));
    await run({ allBranches: true });
    assert.strictEqual(JSON.parse(logOutput1[0]).id, 'spec-one');
    
    // Test second directory
    process.chdir(testDir2);
    const logOutput2 = [];
    t.mock.method(console, 'log', (msg) => logOutput2.push(msg));
    await run({ allBranches: true });
    assert.strictEqual(JSON.parse(logOutput2[0]).id, 'spec-two');
  } finally {
    // CRITICAL: Restore working directory FIRST
    process.chdir(originalCwd);
    
    if (fs.existsSync(testDir1)) {
      fs.rmSync(testDir1, { recursive: true, force: true });
    }
    if (fs.existsSync(testDir2)) {
      fs.rmSync(testDir2, { recursive: true, force: true });
    }
  }
});

test('CLI shows appropriate error when no specs directory exists', async (t) => {
  const originalCwd = process.cwd();
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'cli-test-'));
  
  try {
    // Don't create specs directory - should get an error
    process.chdir(testDir);
    
    const errorOutput = [];
    t.mock.method(console, 'error', (msg) => errorOutput.push(msg));
    
    await run({ allBranches: true });
    
    assert.ok(errorOutput.length > 0);
    assert.ok(errorOutput[0].includes('No specs found'));
  } finally {
    // CRITICAL: Restore working directory FIRST
    process.chdir(originalCwd);
    
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }
  }
});

test('CLI handles nested directory structures', async (t) => {
  const originalCwd = process.cwd();
  const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'cli-test-'));
  const specsDir = path.join(testDir, 'specs');
  
  try {
    // Create nested structure
    fs.mkdirSync(specsDir, { recursive: true });
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
    
    process.chdir(testDir);
    
    const logOutput = [];
    t.mock.method(console, 'log', (msg) => logOutput.push(msg));
    
    await run({ allBranches: true });
    
    assert.strictEqual(logOutput.length, 1);
    const output = JSON.parse(logOutput[0]);
    assert.strictEqual(output.id, 'child-assertion');
  } finally {
    // CRITICAL: Restore working directory FIRST
    process.chdir(originalCwd);
    
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }
  }
});
