import { describe, it, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert';
import { parseAllSpecs, run } from '../index.js';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import os from 'os';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

describe('CLI Reads Local Specs', () => {
  let originalCwd;
  let tempDir;
  let capturedOutput;
  let originalLog;

  beforeEach(() => {
    // Save original working directory
    originalCwd = process.cwd();
    
    // Create a unique temporary directory
    tempDir = path.join(os.tmpdir(), `spekk-test-${Date.now()}`);
    fs.mkdirSync(tempDir, { recursive: true });
    
    // Create a test spec structure in temp directory
    const specsDir = path.join(tempDir, 'specs');
    const testSpecDir = path.join(specsDir, 'test-spec');
    const assertionsDir = path.join(testSpecDir, 'assertions');
    
    fs.mkdirSync(specsDir, { recursive: true });
    fs.mkdirSync(testSpecDir, { recursive: true });
    fs.mkdirSync(assertionsDir, { recursive: true });
    
    // Create a test spec file
    const specContent = `---
id: test-spec
created: 2026-01-28T10:00:00Z
priority: 1
status: in_progress
---

# Test Spec

This is a test spec in the temporary directory.`;
    
    fs.writeFileSync(path.join(testSpecDir, 'test-spec.md'), specContent);
    
    // Create a test assertion file
    const assertionContent = `---
id: test-assertion
parent: test-spec
created: 2026-01-28T10:00:00Z
priority: 1
status: not_started
---

# Test Assertion

This is a test assertion in the temporary directory.`;
    
    fs.writeFileSync(path.join(assertionsDir, 'test-assertion.md'), assertionContent);
    
    // Capture console.log output
    capturedOutput = [];
    originalLog = console.log;
    console.log = (...args) => {
      capturedOutput.push(args.join(' '));
    };
  });

  afterEach(() => {
    // Restore original working directory
    process.chdir(originalCwd);
    
    // Clean up temp directory
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
    
    // Restore console.log
    console.log = originalLog;
  });

  it('should read specs from current working directory when no specsDirectory provided', () => {
    // Change to the temp directory
    process.chdir(tempDir);
    
    // Call parseAllSpecs without arguments (should use process.cwd())
    const result = parseAllSpecs();
    
    // Should find our test spec and assertion
    assert.strictEqual(result.specs.length, 1);
    assert.strictEqual(result.specs[0].id, 'test-spec');
    assert.strictEqual(result.assertions.length, 1);
    assert.strictEqual(result.assertions[0].id, 'test-assertion');
  });

  it('should use process.cwd() for spekk next command', () => {
    // Change to the temp directory
    process.chdir(tempDir);
    
    // Run the parser (simulates 'spekk next')
    run();
    
    // Parse the JSON output
    assert.strictEqual(capturedOutput.length, 1);
    const output = JSON.parse(capturedOutput[0]);
    
    // Should return our test assertion from the temp directory
    assert.strictEqual(output.type, 'assertion');
    assert.strictEqual(output.id, 'test-assertion');
    assert.strictEqual(output.parent, 'test-spec');
  });

  it('should read different specs when run from different directories', () => {
    // First, run from temp directory
    process.chdir(tempDir);
    run();
    
    assert.strictEqual(capturedOutput.length, 1);
    const tempDirOutput = JSON.parse(capturedOutput[0]);
    assert.strictEqual(tempDirOutput.id, 'test-assertion');
    
    // Clear captured output
    capturedOutput.length = 0;
    
    // Now run from original directory (spekk-cli installation)
    process.chdir(originalCwd);
    run();
    
    assert.strictEqual(capturedOutput.length, 1);
    const installDirOutput = JSON.parse(capturedOutput[0]);
    
    // Should get different results from different directories
    assert.notStrictEqual(installDirOutput.id, 'test-assertion');
  });

  it('should show appropriate error when no specs directory exists', () => {
    // Create a temp directory with no specs folder
    const emptyDir = path.join(os.tmpdir(), `spekk-empty-${Date.now()}`);
    fs.mkdirSync(emptyDir, { recursive: true });
    
    try {
      // Change to the empty directory
      process.chdir(emptyDir);
      
      // Run the parser
      run();
      
      // Parse the JSON output
      assert.strictEqual(capturedOutput.length, 1);
      const output = JSON.parse(capturedOutput[0]);
      
      // Should indicate no specifications found
      assert.strictEqual(output.status, 'empty');
      assert.match(output.message, /no specifications found/i);
    } finally {
      // Clean up
      process.chdir(originalCwd);
      fs.rmSync(emptyDir, { recursive: true, force: true });
    }
  });

  it('should handle nested directory structures correctly', () => {
    // Create a nested project structure
    const projectDir = path.join(tempDir, 'my-project');
    const projectSpecsDir = path.join(projectDir, 'specs');
    const nestedSpecDir = path.join(projectSpecsDir, 'nested-spec');
    const nestedAssertionsDir = path.join(nestedSpecDir, 'assertions');
    
    fs.mkdirSync(projectDir, { recursive: true });
    fs.mkdirSync(projectSpecsDir, { recursive: true });
    fs.mkdirSync(nestedSpecDir, { recursive: true });
    fs.mkdirSync(nestedAssertionsDir, { recursive: true });
    
    // Create a nested spec
    const nestedSpecContent = `---
id: nested-spec
created: 2026-01-28T11:00:00Z
priority: 1
status: not_started
---

# Nested Spec`;
    
    fs.writeFileSync(path.join(nestedSpecDir, 'nested-spec.md'), nestedSpecContent);
    
    // Create a nested assertion
    const nestedAssertionContent = `---
id: nested-assertion
parent: nested-spec
created: 2026-01-28T11:00:00Z
priority: 1
status: not_started
---

# Nested Assertion`;
    
    fs.writeFileSync(path.join(nestedAssertionsDir, 'nested-assertion.md'), nestedAssertionContent);
    
    // Change to the nested project directory
    process.chdir(projectDir);
    
    // Run the parser
    run();
    
    // Parse the JSON output
    assert.strictEqual(capturedOutput.length, 1);
    const output = JSON.parse(capturedOutput[0]);
    
    // Should find the nested assertion
    assert.strictEqual(output.type, 'assertion');
    assert.strictEqual(output.id, 'nested-assertion');
    assert.strictEqual(output.parent, 'nested-spec');
  });
});