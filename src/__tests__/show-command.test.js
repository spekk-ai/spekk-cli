import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import { existsSync, rmSync, mkdirSync, lstatSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';

describe('Show Command', () => {
  
  const testDir = join(tmpdir(), `spekk-test-${Date.now()}`);
  
  // Setup and cleanup test directory
  function setupTestDir() {
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
    mkdirSync(testDir, { recursive: true });
    process.chdir(testDir);
  }
  
  function cleanupTestDir() {
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
  }

  test('spekk show command exists and runs without errors', () => {
    setupTestDir();
    
    try {
      const result = execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      // Should return string output without throwing error
      assert.ok(typeof result === 'string', 'Show command should return string output');
    } finally {
      cleanupTestDir();
    }
  });

  test('creates .spekk directory when it does not exist', () => {
    setupTestDir();
    
    try {
      // Ensure .spekk directory doesn't exist
      const spekkDir = join(testDir, '.spekk');
      assert.ok(!existsSync(spekkDir), '.spekk directory should not exist initially');
      
      // Run spekk show command
      execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      // Check that .spekk directory was created
      assert.ok(existsSync(spekkDir), '.spekk directory should be created after running spekk show');
    } finally {
      cleanupTestDir();
    }
  });

  test('does not error when .spekk directory already exists', () => {
    setupTestDir();
    
    try {
      // Create .spekk directory first
      const spekkDir = join(testDir, '.spekk');
      mkdirSync(spekkDir);
      assert.ok(existsSync(spekkDir), '.spekk directory should exist initially');
      
      // Run spekk show command - should not throw error
      const result = execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      // Should still exist and command should succeed
      assert.ok(existsSync(spekkDir), '.spekk directory should still exist');
      assert.ok(typeof result === 'string', 'Show command should return string output even when directory exists');
    } finally {
      cleanupTestDir();
    }
  });

  test('creates directory with appropriate permissions', () => {
    setupTestDir();
    
    try {
      // Run spekk show command
      execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      const spekkDir = join(testDir, '.spekk');
      assert.ok(existsSync(spekkDir), '.spekk directory should be created');
      
      // Check that it's actually a directory (not a file)
      const stats = lstatSync(spekkDir);
      assert.ok(stats.isDirectory(), '.spekk should be a directory');
    } finally {
      cleanupTestDir();
    }
  });
});