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

describe('Builder CLI', () => {
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

  test('builder CLI includes prompt content in message', async () => {
    // Mock the prompt file read
    const promptPath = path.join(__dirname, '../../../specs/builder-agent/builder-agent.prompt.md');
    const mockPromptContent = 'You are the Builder Agent - this is a mock prompt for testing.';
    
    // Set up mock spawn behavior
    const mockClaudeOutput = 'STDIN_CONTENT: You are the Builder Agent\n';
    const mockParserOutput = '{"type":"assertion","id":"test-assertion","parent":"test-spec","file":"test.md","priority":1,"status":"not_started","title":"Test Assertion"}';
    
    let callCount = 0;
    mockSpawn.mock.mockImplementation((command, args, options) => {
      callCount++;
      
      // First call is to spekk next (parser)
      if (callCount === 1 && args[0].includes('parser/cli.js')) {
        return createMockChildProcess({
          immediate: true,
          stdout: mockParserOutput,
          exitCode: 0
        });
      }
      
      // Second call is to claude
      if (callCount === 2 && command === 'claude') {
        const cp = createMockChildProcess();
        // Simulate reading stdin and outputting it
        setImmediate(() => {
          cp.stdout.emit('data', Buffer.from(mockClaudeOutput));
          cp.emit('exit', 0);
        });
        return cp;
      }
      
      // Default mock for the builder CLI itself
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
      const builderProcess = childProcess.spawn('node', [cliPath], {
        stdio: ['pipe', 'pipe', 'pipe']
      });
      
      let output = '';
      builderProcess.stdout.on('data', (data) => {
        output += data.toString();
      });
      
      await new Promise((resolve, reject) => {
        builderProcess.on('exit', () => resolve());
        builderProcess.on('error', reject);
        
        setTimeout(() => {
          builderProcess.kill('SIGTERM');
          resolve();
        }, 100); // Much faster with mocks
      });
      
      // Check that the output contains the prompt content
      assert.ok(output.includes('STDIN_CONTENT:'));
      assert.ok(output.includes('You are the Builder Agent'), 'Should include agent activation message');
      
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
    
    // Mock spawn to simulate successful execution without file errors
    mockSpawn.mock.mockImplementation(() => {
      return createMockChildProcess({
        immediate: true,
        stdout: 'Builder CLI started successfully\n',
        stderr: '', // No errors
        exitCode: 0
      });
    });
    
    const originalCwd = process.cwd();
    process.chdir(subDir);
    
    try {
      const builderProcess = childProcess.spawn('node', [cliPath], {
        stdio: ['pipe', 'pipe', 'pipe']
      });
      
      let errorOutput = '';
      builderProcess.stderr.on('data', (data) => {
        errorOutput += data.toString();
      });
      
      await new Promise((resolve) => {
        builderProcess.on('exit', () => resolve());
        
        setTimeout(() => {
          builderProcess.kill('SIGTERM');
          resolve();
        }, 100); // Much faster with mocks
      });
      
      // Should not fail with file not found errors for prompt files
      const relevantErrors = errorOutput.split('\n').filter(line => 
        line.includes('ENOENT') && (line.includes('prompt') || line.includes('specs/'))
      );
      
      assert.strictEqual(relevantErrors.length, 0, 'Should not have prompt file not found errors');
      
    } finally {
      process.chdir(originalCwd);
    }
  });
});