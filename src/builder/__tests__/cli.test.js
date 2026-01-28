import { test, describe, before, after } from 'node:test';
import assert from 'node:assert';
import { spawn } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

describe('Builder CLI', () => {
  let tempDir;
  
  before(() => {
    // Create a temporary directory to test from
    tempDir = fs.mkdtempSync(path.join(process.cwd(), 'temp-test-'));
  });
  
  after(() => {
    // Clean up temporary directory
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  test('builder CLI includes prompt content in message', async () => {
    // Run builder CLI and capture the message it sends to Claude Code
    const cliPath = path.join(__dirname, '../cli.js');
    
    // Mock Claude Code by creating a simple script that captures stdin
    const mockClaudePath = path.join(tempDir, 'mock-claude');
    const mockScript = `#!/usr/bin/env node
process.stdin.on('data', (data) => {
  console.log('STDIN_CONTENT:', data.toString());
});
process.stdin.on('end', () => {
  process.exit(0);
});
`;
    
    fs.writeFileSync(mockClaudePath, mockScript);
    fs.chmodSync(mockClaudePath, '755');
    
    // Create a mock parser that returns a simple assertion
    const mockParserPath = path.join(tempDir, 'mock-parser');
    const mockParserScript = `#!/usr/bin/env node
console.log('{"type":"assertion","id":"test-assertion","parent":"test-spec","file":"test.md","priority":1,"status":"not_started","title":"Test Assertion"}');
`;
    fs.writeFileSync(mockParserPath, mockParserScript);
    fs.chmodSync(mockParserPath, '755');
    
    // Update PATH to use our mocks
    const env = { ...process.env };
    env.PATH = `${tempDir}:${env.PATH}`;
    
    // Run the builder CLI from the temp directory
    const originalCwd = process.cwd();
    process.chdir(tempDir);
    
    try {
      const builderProcess = spawn('node', [cliPath], {
        env,
        stdio: ['pipe', 'pipe', 'pipe']
      });
      
      let output = '';
      builderProcess.stdout.on('data', (data) => {
        output += data.toString();
      });
      
      await new Promise((resolve, reject) => {
        builderProcess.on('exit', (code) => {
          resolve();
        });
        
        builderProcess.on('error', reject);
        
        // Give it a moment then kill it
        setTimeout(() => {
          builderProcess.kill('SIGTERM');
          resolve();
        }, 3000);
      });
      
      // Check that the output contains the prompt content, not just a reference
      assert.ok(output.includes('STDIN_CONTENT:'));
      
      // The message should contain the actual builder prompt content
      const promptPath = path.join(__dirname, '../../../specs/builder-agent/builder-agent.prompt.md');
      if (fs.existsSync(promptPath)) {
        // The CLI should include the prompt content, not just tell Claude to read it
        assert.ok(output.includes('You are the Builder Agent'), 'Should include agent activation message');
      }
      
    } finally {
      process.chdir(originalCwd);
    }
  });

  test('builder CLI works from any directory', async () => {
    // Test that the builder CLI can find prompt files even when run from a different directory
    const cliPath = path.join(__dirname, '../cli.js');
    
    // Create a subdirectory to run from
    const subDir = path.join(tempDir, 'subdir');
    fs.mkdirSync(subDir);
    
    const originalCwd = process.cwd();
    process.chdir(subDir);
    
    try {
      const builderProcess = spawn('node', [cliPath], {
        stdio: ['pipe', 'pipe', 'pipe']
      });
      
      let errorOutput = '';
      builderProcess.stderr.on('data', (data) => {
        errorOutput += data.toString();
      });
      
      await new Promise((resolve) => {
        builderProcess.on('exit', () => {
          resolve();
        });
        
        // Give it a moment then kill it
        setTimeout(() => {
          builderProcess.kill('SIGTERM');
          resolve();
        }, 1000);
      });
      
      // Should not fail with file not found errors for prompt files
      // Note: It might fail due to missing parser, but that's a different issue
      const relevantErrors = errorOutput.split('\n').filter(line => 
        line.includes('ENOENT') && (line.includes('prompt') || line.includes('specs/'))
      );
      
      assert.strictEqual(relevantErrors.length, 0, 'Should not have prompt file not found errors');
      
    } finally {
      process.chdir(originalCwd);
    }
  });
});