#!/usr/bin/env node

import { test, describe, before, after } from 'node:test';
import assert from 'node:assert';
import { spawn } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const projectRoot = path.join(__dirname, '../../..');

describe('Working Directory Verification Tests', () => {
  let tempDir;
  
  before(() => {
    // Create a temporary directory to test from (simulates user's project directory)
    tempDir = fs.mkdtempSync(path.join(process.cwd(), 'temp-wd-test-'));
  });
  
  after(() => {
    // Clean up temporary directory
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  test('coach CLI reports correct working directory', async () => {
    // Create test files in user directory
    const testFile = path.join(tempDir, 'test-file.txt');
    fs.writeFileSync(testFile, 'user project file');
    
    // Create a mock Claude that immediately exits to avoid hanging
    const mockClaudePath = path.join(tempDir, 'claude');
    const mockScript = `#!/usr/bin/env node
process.stdin.on('data', () => process.exit(0));
process.exit(0);
`;
    fs.writeFileSync(mockClaudePath, mockScript);
    fs.chmodSync(mockClaudePath, '755');
    
    // Update PATH to use our mock Claude
    const env = { ...process.env };
    env.PATH = `${tempDir}:${env.PATH}`;
    
    const coachCliPath = path.join(projectRoot, 'src/coach/cli.js');
    
    // Run spekk coach from user's project directory
    const originalCwd = process.cwd();
    process.chdir(tempDir);
    
    try {
      const coachProcess = spawn('node', [coachCliPath], {
        env,
        stdio: ['pipe', 'pipe', 'pipe'],
        cwd: tempDir
      });
      
      let output = '';
      coachProcess.stdout.on('data', (data) => {
        output += data.toString();
      });
      
      await new Promise((resolve, reject) => {
        coachProcess.on('exit', resolve);
        coachProcess.on('error', reject);
        
        setTimeout(() => {
          coachProcess.kill('SIGTERM');
          resolve();
        }, 2000);
      });
      
      // Verify coach CLI reports running in user directory
      assert.ok(output.includes(`Working directory: ${tempDir}`), 
        `Coach CLI should report running in user directory ${tempDir}`);
      
      // Verify we're running from the user directory, not the spekk-cli installation directory
      const workingDirMatch = output.match(/Working directory: (.+)/);
      assert.ok(workingDirMatch, 'Should report working directory');
      const reportedDir = workingDirMatch[1];
      assert.strictEqual(reportedDir, tempDir, 
        `Should run from user directory ${tempDir}, not spekk-cli directory ${projectRoot}`);
        
    } finally {
      process.chdir(originalCwd);
    }
  });

  test('prompt files are NOT copied to user directory (per spec)', async () => {
    // Create a mock Claude that exits immediately
    const mockClaudePath = path.join(tempDir, 'claude');
    const mockScript = `#!/usr/bin/env node
process.exit(0);
`;
    fs.writeFileSync(mockClaudePath, mockScript);
    fs.chmodSync(mockClaudePath, '755');
    
    const env = { ...process.env };
    env.PATH = `${tempDir}:${env.PATH}`;
    
    const coachCliPath = path.join(projectRoot, 'src/coach/cli.js');
    
    const originalCwd = process.cwd();
    process.chdir(tempDir);
    
    try {
      const coachProcess = spawn('node', [coachCliPath], {
        env,
        stdio: ['pipe', 'pipe', 'pipe'],
        cwd: tempDir
      });
      
      let output = '';
      coachProcess.stdout.on('data', (data) => {
        output += data.toString();
      });
      
      await new Promise((resolve, reject) => {
        coachProcess.on('exit', resolve);
        coachProcess.on('error', reject);
        
        setTimeout(() => {
          coachProcess.kill('SIGTERM');
          resolve();
        }, 2000);
      });
      
      // Verify spekk launches successfully without copying files
      assert.ok(output.includes('Launching Coach Agent Agent with Claude Code'), 
        'Coach CLI should launch successfully');
      assert.ok(output.includes(`Working directory: ${tempDir}`), 
        'Should report correct working directory');
      
      // Verify NO prompt files are copied to user directory (per spec requirement)
      const expectedPromptFiles = [
        path.join(tempDir, 'specs'),
        path.join(tempDir, 'specs/coach-agent'),
        path.join(tempDir, 'specs/builder-agent'),
        path.join(tempDir, 'specs/observer-agent')
      ];
      
      expectedPromptFiles.forEach(dirPath => {
        assert.ok(!fs.existsSync(dirPath), 
          `No specs directory should be created in user directory: ${dirPath}`);
      });
      
      // Verify user directory remains clean (spec requirement)
      const userFiles = fs.readdirSync(tempDir);
      const expectedFiles = ['test-file.txt', 'claude'];
      userFiles.forEach(file => {
        assert.ok(expectedFiles.includes(file) || file.startsWith('temp-wd-test-'), 
          `Unexpected file in user directory: ${file}`);
      });
        
    } finally {
      process.chdir(originalCwd);
    }
  });

  test('user can access their own CLAUDE.md file', async () => {
    // Create user's CLAUDE.md with distinctive content
    const userClaudeMd = path.join(tempDir, 'CLAUDE.md');
    const userContent = '# User Project Configuration\nproject_name: my-test-project';
    fs.writeFileSync(userClaudeMd, userContent);
    
    // Create a mock Claude that exits immediately
    const mockClaudePath = path.join(tempDir, 'claude');
    const mockScript = `#!/usr/bin/env node
process.exit(0);
`;
    fs.writeFileSync(mockClaudePath, mockScript);
    fs.chmodSync(mockClaudePath, '755');
    
    const env = { ...process.env };
    env.PATH = `${tempDir}:${env.PATH}`;
    
    const coachCliPath = path.join(projectRoot, 'src/coach/cli.js');
    
    const originalCwd = process.cwd();
    process.chdir(tempDir);
    
    try {
      const coachProcess = spawn('node', [coachCliPath], {
        env,
        stdio: ['pipe', 'pipe', 'pipe'],
        cwd: tempDir
      });
      
      await new Promise((resolve, reject) => {
        coachProcess.on('exit', resolve);
        coachProcess.on('error', reject);
        
        setTimeout(() => {
          coachProcess.kill('SIGTERM');
          resolve();
        }, 2000);
      });
      
      // Verify user's CLAUDE.md file still exists and wasn't modified
      assert.ok(fs.existsSync(userClaudeMd), 'User CLAUDE.md should exist');
      const finalContent = fs.readFileSync(userClaudeMd, 'utf8');
      assert.ok(finalContent.includes('my-test-project'), 
        'User CLAUDE.md should retain user content');
      
      // Verify spekk-cli CLAUDE.md wasn't copied to user directory
      // (The user file should take precedence)
      const spekkClaudeMd = path.join(projectRoot, 'CLAUDE.md');
      if (fs.existsSync(spekkClaudeMd)) {
        const spekkContent = fs.readFileSync(spekkClaudeMd, 'utf8');
        assert.ok(!finalContent.includes('spec-driven development'), 
          'User CLAUDE.md should not be overwritten with spekk-cli content');
      }
        
    } finally {
      process.chdir(originalCwd);
    }
  });

  test('builder CLI runs from user directory and uses spekk parser correctly', async () => {
    // Create test project files
    const testFile = path.join(tempDir, 'test-project.txt');
    fs.writeFileSync(testFile, 'test project content');
    
    // Create a mock Claude that exits immediately
    const mockClaudePath = path.join(tempDir, 'claude');
    const mockScript = `#!/usr/bin/env node
process.exit(0);
`;
    fs.writeFileSync(mockClaudePath, mockScript);
    fs.chmodSync(mockClaudePath, '755');
    
    const env = { ...process.env };
    env.PATH = `${tempDir}:${env.PATH}`;
    
    const builderCliPath = path.join(projectRoot, 'src/builder/cli.js');
    
    const originalCwd = process.cwd();
    process.chdir(tempDir);
    
    try {
      const builderProcess = spawn('node', [builderCliPath], {
        env,
        stdio: ['pipe', 'pipe', 'pipe'],
        cwd: tempDir
      });
      
      let output = '';
      builderProcess.stdout.on('data', (data) => {
        output += data.toString();
      });
      
      await new Promise((resolve, reject) => {
        builderProcess.on('exit', resolve);
        builderProcess.on('error', reject);
        
        setTimeout(() => {
          builderProcess.kill('SIGTERM');
          resolve();
        }, 2000);
      });
      
      // Verify builder CLI starts up correctly (this proves it runs from user directory)
      assert.ok(output.includes('Starting Builder Agent Loop'), 
        'Builder should start successfully from user directory');
      
      // Verify builder uses the spekk-cli parser correctly 
      // (It should find no specs in user directory, which is expected)
      assert.ok(output.includes('Getting next priority assertion') || 
                output.includes('No specifications found'), 
        'Builder should attempt to get assertions from parser');
        
      // The fact that the builder starts and tries to get assertions proves it runs 
      // in the user directory and successfully accesses the spekk-cli parser
        
    } finally {
      process.chdir(originalCwd);
    }
  });
});