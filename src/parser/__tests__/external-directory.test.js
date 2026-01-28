import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { run } from '../index.js';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

describe('External Directory Functionality', () => {
  let originalCwd;
  let tempDir;
  let capturedOutput;

  beforeEach(() => {
    originalCwd = process.cwd();
    tempDir = path.join(__dirname, '../../../temp-test-hVn1Qd');
    
    // Create a temporary directory to simulate external usage
    if (!fs.existsSync(tempDir)) {
      fs.mkdirSync(tempDir, { recursive: true });
    }
    
    // Change to the temp directory
    process.chdir(tempDir);
    
    // Capture console.log output
    capturedOutput = [];
    const originalLog = console.log;
    console.log = (...args) => {
      capturedOutput.push(args.join(' '));
      originalLog(...args);
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
    console.log = console.log.__original || console.log;
  });

  it('should find next assertion when run from external directory', () => {
    // Run the parser from external directory
    run();
    
    // Parse the JSON output
    expect(capturedOutput).toHaveLength(1);
    const output = JSON.parse(capturedOutput[0]);
    
    // Should successfully return an assertion
    expect(output.type).toBe('assertion');
    expect(output.id).toBeDefined();
    expect(output.file).toMatch(/^specs\//);
    expect(output.title).toBeDefined();
  });

  it('should list all specs when run with --all flag from external directory', () => {
    // Run the parser with --all flag from external directory
    run({ all: true });
    
    // Parse the JSON output (find the first valid JSON output)
    expect(capturedOutput.length).toBeGreaterThanOrEqual(1);
    const jsonOutput = capturedOutput.find(output => {
      try {
        const parsed = JSON.parse(output);
        return parsed.type === 'hierarchy';
      } catch {
        return false;
      }
    });
    expect(jsonOutput).toBeDefined();
    const output = JSON.parse(jsonOutput);
    
    // Should successfully return hierarchy
    expect(output.type).toBe('hierarchy');
    expect(output.specs).toBeDefined();
    expect(Array.isArray(output.specs)).toBe(true);
    expect(output.specs.length).toBeGreaterThan(0);
    
    // Each spec should have assertions
    output.specs.forEach(spec => {
      expect(spec.id).toBeDefined();
      expect(spec.title).toBeDefined();
      expect(spec.file).toMatch(/^specs\//);
      expect(Array.isArray(spec.assertions)).toBe(true);
    });
  });

  it('should produce identical output whether run from installation or external directory', () => {
    // Run from external directory
    run();
    const externalOutput = JSON.parse(capturedOutput[0]);
    
    // Clear captured output
    capturedOutput.length = 0;
    
    // Change back to installation directory
    process.chdir(originalCwd);
    
    // Run from installation directory
    run();
    const installationOutput = JSON.parse(capturedOutput[0]);
    
    // Outputs should be identical
    expect(externalOutput).toEqual(installationOutput);
  });

  it('should not produce "directory not found" errors from external directory', () => {
    // Run from external directory
    run();
    
    // Parse the JSON output (find the first valid JSON output)
    expect(capturedOutput.length).toBeGreaterThanOrEqual(1);
    const jsonOutput = capturedOutput.find(output => {
      try {
        return JSON.parse(output);
      } catch {
        return false;
      }
    });
    expect(jsonOutput).toBeDefined();
    const output = JSON.parse(jsonOutput);
    
    // Should not be an error response
    expect(output.error).toBeUndefined();
    if (output.message) {
      expect(output.message).not.toMatch(/directory not found/i);
      expect(output.message).not.toMatch(/no specs found/i);
    }
  });
});