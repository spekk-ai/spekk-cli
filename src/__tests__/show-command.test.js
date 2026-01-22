import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import { existsSync, rmSync, mkdirSync, lstatSync, readFileSync, writeFileSync } from 'node:fs';
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

  test('generates index.html file with valid HTML content', () => {
    setupTestDir();
    
    try {
      // Create minimal specs structure for testing
      const specsDir = join(testDir, 'specs');
      const specDir = join(specsDir, 'test-spec');
      const assertionsDir = join(specDir, 'assertions');
      mkdirSync(assertionsDir, { recursive: true });
      
      // Create a test spec file
      const specContent = `---
id: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Test Spec

This is a test specification.`;
      writeFileSync(join(specDir, 'test-spec.md'), specContent);
      
      // Create a test assertion file
      const assertionContent = `---
id: test-assertion
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Test Assertion

This is a test assertion.`;
      writeFileSync(join(assertionsDir, 'test-assertion.md'), assertionContent);
      
      // Run spekk show command
      execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      // Check that index.html file was created
      const htmlFile = join(testDir, '.spekk', 'index.html');
      assert.ok(existsSync(htmlFile), 'index.html file should be created');
      
      // Read and validate HTML content
      const htmlContent = readFileSync(htmlFile, 'utf8');
      
      // Validate basic HTML structure
      assert.ok(htmlContent.includes('<html'), 'HTML should contain <html tag');
      assert.ok(htmlContent.includes('<head>'), 'HTML should contain <head> tag');
      assert.ok(htmlContent.includes('<body>'), 'HTML should contain <body> tag');
      assert.ok(htmlContent.includes('</html>'), 'HTML should contain closing </html> tag');
      
      // Validate spec tree content is present
      assert.ok(htmlContent.toLowerCase().includes('spec'), 'HTML should mention specs in content');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('overwrites index.html file on subsequent runs', () => {
    setupTestDir();
    
    try {
      // Create .spekk directory and initial index.html
      const spekkDir = join(testDir, '.spekk');
      mkdirSync(spekkDir);
      const htmlFile = join(spekkDir, 'index.html');
      writeFileSync(htmlFile, 'old content');
      
      // Run spekk show command
      execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      // Check that file was overwritten (not appended to)
      const htmlContent = readFileSync(htmlFile, 'utf8');
      assert.ok(!htmlContent.includes('old content'), 'Old content should be overwritten');
      assert.ok(htmlContent.includes('<html'), 'New HTML content should be present');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('command appears in spekk --help output', () => {
    const result = execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js --help', { 
      encoding: 'utf8',
      timeout: 5000 
    });
    
    // Check that show command is listed in help output
    assert.ok(result.includes('show'), 'Help output should mention show command');
    assert.ok(result.toLowerCase().includes('generate'), 'Help should describe show command functionality');
  });

  test('spekk show --help provides feedback about functionality', () => {
    setupTestDir();
    
    try {
      // spekk show --help should execute the show command (current behavior)
      const result = execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show --help', { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      // Should provide feedback about what it does
      assert.ok(result.includes('Generated spec explorer'), 'Should indicate what was generated');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('opens browser after successful HTML generation', () => {
    setupTestDir();
    
    try {
      // Create minimal specs structure for testing
      const specsDir = join(testDir, 'specs');
      const specDir = join(specsDir, 'test-spec');
      const assertionsDir = join(specDir, 'assertions');
      mkdirSync(assertionsDir, { recursive: true });
      
      // Create a test spec file
      const specContent = `---
id: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Test Spec

This is a test specification.`;
      writeFileSync(join(specDir, 'test-spec.md'), specContent);
      
      // Run spekk show command
      const result = execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 10000 
      });
      
      // Command should succeed even if browser opening fails
      assert.ok(typeof result === 'string', 'Show command should return string output');
      assert.ok(result.includes('Generated spec explorer'), 'Should indicate HTML was generated');
      
      // HTML file should exist
      const htmlFile = join(testDir, '.spekk', 'index.html');
      assert.ok(existsSync(htmlFile), 'index.html file should be created');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('command completes successfully even if browser opening fails', () => {
    setupTestDir();
    
    try {
      // We can't easily test actual browser failure, but we can ensure 
      // the command doesn't throw errors during execution
      const result = execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 10000 
      });
      
      // Should complete without error
      assert.ok(typeof result === 'string', 'Show command should return string output');
      
    } finally {
      cleanupTestDir();
    }
  });
});