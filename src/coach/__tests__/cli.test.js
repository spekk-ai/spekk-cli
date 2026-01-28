import { test, describe, before, after } from 'node:test';
import assert from 'node:assert';
import { spawn } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

describe('Coach CLI', () => {
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

  test('coach CLI includes prompt content in message', async () => {
    // Run coach CLI and capture the message it sends to Claude Code
    const cliPath = path.join(__dirname, '../cli.js');
    
    // Mock Claude Code by creating a simple script that captures stdin
    const mockClaudePath = path.join(tempDir, 'claude');
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
    
    // Update PATH to use our mock
    const env = { ...process.env };
    env.PATH = `${tempDir}:${env.PATH}`;
    
    // Run the coach CLI from the temp directory
    const originalCwd = process.cwd();
    process.chdir(tempDir);
    
    try {
      const coachProcess = spawn('node', [cliPath], {
        env,
        stdio: ['pipe', 'pipe', 'pipe']
      });
      
      let output = '';
      coachProcess.stdout.on('data', (data) => {
        output += data.toString();
      });
      
      await new Promise((resolve, reject) => {
        coachProcess.on('exit', (code) => {
          if (code === 0) {
            resolve();
          } else {
            reject(new Error(`Coach CLI exited with code ${code}`));
          }
        });
        
        coachProcess.on('error', reject);
        
        // Give it a moment then kill it
        setTimeout(() => {
          coachProcess.kill('SIGTERM');
          resolve();
        }, 2000);
      });
      
      // Check that the output contains the prompt content, not just a reference
      assert.ok(output.includes('STDIN_CONTENT:'));
      
      // The message should contain the actual coach prompt content
      const promptPath = path.join(__dirname, '../../../specs/coach-agent/coach-agent.prompt.md');
      if (fs.existsSync(promptPath)) {
        const promptContent = fs.readFileSync(promptPath, 'utf8');
        // The CLI should include the prompt content, not just tell Claude to read it
        assert.ok(output.includes('You are the Coach Agent'), 'Should include agent activation message');
      }
      
    } finally {
      process.chdir(originalCwd);
    }
  });

  test('coach CLI works from any directory', async () => {
    // Test that the coach CLI can find prompt files even when run from a different directory
    const cliPath = path.join(__dirname, '../cli.js');
    
    // Create a subdirectory to run from
    const subDir = path.join(tempDir, 'subdir');
    fs.mkdirSync(subDir);
    
    const originalCwd = process.cwd();
    process.chdir(subDir);
    
    try {
      const coachProcess = spawn('node', [cliPath], {
        stdio: ['pipe', 'pipe', 'pipe']
      });
      
      let errorOutput = '';
      coachProcess.stderr.on('data', (data) => {
        errorOutput += data.toString();
      });
      
      await new Promise((resolve) => {
        coachProcess.on('exit', () => {
          resolve();
        });
        
        // Give it a moment then kill it
        setTimeout(() => {
          coachProcess.kill('SIGTERM');
          resolve();
        }, 1000);
      });
      
      // Should not fail with file not found errors
      assert.ok(!errorOutput.includes('ENOENT'), 'Should not have file not found errors');
      assert.ok(!errorOutput.includes('Cannot find'), 'Should not have module not found errors');
      
    } finally {
      process.chdir(originalCwd);
    }
  });
});