import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import { existsSync, rmSync, mkdirSync, writeFileSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';

describe('Clicking Shows Details Panel', () => {
  
  const testDir = join(tmpdir(), `spekk-test-${Date.now()}`);
  
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

  function createTestSpec() {
    const specsDir = join(testDir, 'specs');
    const specDir = join(specsDir, 'test-spec');
    const assertionsDir = join(specDir, 'assertions');
    mkdirSync(assertionsDir, { recursive: true });
    
    const specContent = `---
id: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Test Spec

This is a test specification for testing.`;
    writeFileSync(join(specDir, 'test-spec.md'), specContent);
    
    const assertionContent = `---
id: test-assertion
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Test Assertion

This is a test assertion for testing.`;
    writeFileSync(join(assertionsDir, 'test-assertion.md'), assertionContent);
  }

  test('HTML contains showDetail JavaScript function', () => {
    setupTestDir();
    
    try {
      createTestSpec();
      
      execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');
      
      // Verify showDetail function exists
      assert.ok(htmlContent.includes('function showDetail('), 'HTML should contain showDetail function');
      
      // Verify function accepts event parameter
      assert.ok(htmlContent.includes('function showDetail(id, type, event'), 'showDetail should accept event parameter');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('onclick handlers include event parameter for assertions', () => {
    setupTestDir();
    
    try {
      createTestSpec();
      
      execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');
      
      // Verify onclick handlers pass event parameter
      assert.ok(htmlContent.includes("'assertion', event)"), 'onclick handlers should pass event parameter');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('HTML includes CSS for selected state styling', () => {
    setupTestDir();
    
    try {
      createTestSpec();
      
      execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');
      
      // Verify CSS for selected assertion items
      assert.ok(htmlContent.includes('.assertion-item.selected'), 'HTML should contain CSS for selected assertion items');
      
      // Verify CSS for selected spec headers
      assert.ok(htmlContent.includes('.spec-header.selected'), 'HTML should contain CSS for selected spec headers');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('HTML contains detail panel structure', () => {
    setupTestDir();
    
    try {
      createTestSpec();
      
      execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');
      
      // Verify detail panel exists
      assert.ok(htmlContent.includes('class="detail-panel"'), 'HTML should contain detail panel');
      
      // Verify detail content structure
      assert.ok(htmlContent.includes('class="detail-content"'), 'HTML should contain detail content elements');
      
      // Verify empty state
      assert.ok(htmlContent.includes('id="empty-state"'), 'HTML should contain empty state element');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('onclick handlers are attached to assertion items', () => {
    setupTestDir();
    
    try {
      createTestSpec();
      
      execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');
      
      // Verify assertion items have onclick handlers
      assert.ok(htmlContent.includes('class="assertion-item" onclick='), 'Assertion items should have onclick handlers');
      
    } finally {
      cleanupTestDir();
    }
  });
});