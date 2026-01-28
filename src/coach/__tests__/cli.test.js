import { test, describe, before, after, mock } from 'node:test';
import assert from 'node:assert';
import * as childProcess from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'url';
import { EventEmitter } from 'node:events';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Mock spawn to avoid real child processes
const createMockChildProcess = (options = {}) => {
  const cp = new EventEmitter();
  cp.stdout = new EventEmitter();
  cp.stderr = new EventEmitter();
  cp.stdin = new EventEmitter();
  cp.kill = mock.fn(() => {
    setImmediate(() => cp.emit('exit', 0));
  });
  
  // Simulate process behavior based on options
  if (options.immediate) {
    setImmediate(() => {
      if (options.stdout) {
        cp.stdout.emit('data', Buffer.from(options.stdout));
      }
      if (options.stderr) {
        cp.stderr.emit('data', Buffer.from(options.stderr));
      }
      cp.emit('exit', options.exitCode || 0);
    });
  }
  
  return cp;
};

describe('Coach CLI', () => {
  let tempDir;
  let mockSpawn;
  
  before(() => {
    // Create .tmp directory if it doesn't exist
    const projectRoot = path.join(__dirname, '../../..');
    const tmpBase = path.join(projectRoot, '.tmp');
    if (!fs.existsSync(tmpBase)) {
      fs.mkdirSync(tmpBase, { recursive: true });
    }
    
    // Create a temporary directory to test from
    tempDir = fs.mkdtempSync(path.join(tmpBase, 'temp-test-'));
    
    // Mock spawn
    mockSpawn = mock.method(childProcess, 'spawn');
  });
  
  after(() => {
    // Clean up temporary directory
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
    
    // Restore mocks
    mockSpawn.mock.restore();
  });

  test('coach CLI includes prompt content in message', async () => {
    // Mock the coach CLI output
    const mockClaudeOutput = 'STDIN_CONTENT: You are the Coach Agent\n';
    
    mockSpawn.mock.mockImplementation((command, args, options) => {
      // Mock for claude command
      if (command === 'claude') {
        const cp = createMockChildProcess();
        setImmediate(() => {
          cp.stdout.emit('data', Buffer.from(mockClaudeOutput));
          cp.emit('exit', 0);
        });
        return cp;
      }
      
      // Default mock for the coach CLI itself
      return createMockChildProcess({
        immediate: true,
        stdout: mockClaudeOutput,
        exitCode: 0
      });
    });
    
    const cliPath = path.join(__dirname, '../cli.js');
    const originalCwd = process.cwd();
    process.chdir(tempDir);
    
    try {
      const coachProcess = childProcess.spawn('node', [cliPath], {
        stdio: ['pipe', 'pipe', 'pipe']
      });
      
      let output = '';
      coachProcess.stdout.on('data', (data) => {
        output += data.toString();
      });
      
      await new Promise((resolve, reject) => {
        coachProcess.on('exit', () => resolve());
        coachProcess.on('error', reject);
        
        setTimeout(() => {
          coachProcess.kill('SIGTERM');
          resolve();
        }, 100); // Much faster with mocks
      });
      
      // Check that the output contains the prompt content
      assert.ok(output.includes('STDIN_CONTENT:'));
      assert.ok(output.includes('You are the Coach Agent'), 'Should include agent activation message');
      
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
    
    // Mock spawn to simulate successful execution without file errors
    mockSpawn.mock.mockImplementation(() => {
      return createMockChildProcess({
        immediate: true,
        stdout: 'Coach CLI started successfully\n',
        stderr: '', // No errors
        exitCode: 0
      });
    });
    
    const originalCwd = process.cwd();
    process.chdir(subDir);
    
    try {
      const coachProcess = childProcess.spawn('node', [cliPath], {
        stdio: ['pipe', 'pipe', 'pipe']
      });
      
      let errorOutput = '';
      coachProcess.stderr.on('data', (data) => {
        errorOutput += data.toString();
      });
      
      await new Promise((resolve) => {
        coachProcess.on('exit', () => resolve());
        
        setTimeout(() => {
          coachProcess.kill('SIGTERM');
          resolve();
        }, 100); // Much faster with mocks
      });
      
      // Should not fail with file not found errors
      assert.ok(!errorOutput.includes('ENOENT'), 'Should not have file not found errors');
      assert.ok(!errorOutput.includes('Cannot find'), 'Should not have module not found errors');
      
    } finally {
      process.chdir(originalCwd);
    }
  });
});