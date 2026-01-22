import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import { existsSync, rmSync, mkdirSync, writeFileSync, readFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { tmpdir } from 'node:os';

describe('Clicking Shows Details Panel', () => {
  
  const testDir = join(tmpdir(), `spekk-test-${Date.now()}`);
  const __dirname = dirname(fileURLToPath(import.meta.url));
  const projectRoot = join(__dirname, '../..');
  
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

  test('HTML uses event delegation with document listener', () => {
    setupTestDir();
    
    try {
      createTestSpec();
      
      execSync(`node "${join(projectRoot, 'bin/spekk.js')}" show`, { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');
      
      // Verify event delegation is implemented
      assert.ok(htmlContent.includes('document.addEventListener('), 'Should use document event listener');
      assert.ok(htmlContent.includes("'click'"), 'Should listen for click events');
      assert.ok(htmlContent.includes('data-action'), 'Should check data-action attribute');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('assertion items have data attributes for event delegation', () => {
    setupTestDir();
    
    try {
      createTestSpec();
      
      execSync(`node "${join(projectRoot, 'bin/spekk.js')}" show`, { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');
      
      // Verify assertion items have data attributes
      assert.ok(htmlContent.includes('data-action="show-detail"'), 'assertion items should have data-action attribute');
      assert.ok(htmlContent.includes('data-assertion-id='), 'assertion items should have data-assertion-id attribute');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('HTML includes CSS for selected state styling', () => {
    setupTestDir();
    
    try {
      createTestSpec();
      
      execSync(`node "${join(projectRoot, 'bin/spekk.js')}" show`, { 
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
      
      execSync(`node "${join(projectRoot, 'bin/spekk.js')}" show`, { 
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

  test('no inline onclick handlers are used for detail panel functionality', () => {
    setupTestDir();
    
    try {
      createTestSpec();
      
      execSync(`node "${join(projectRoot, 'bin/spekk.js')}" show`, { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');
      
      // Verify assertion items do NOT have inline onclick handlers for detail functionality
      assert.ok(!htmlContent.includes('onclick="showDetail'), 'Should not use inline onclick handlers for showDetail function');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('event delegation is set up with document listener', () => {
    setupTestDir();
    
    try {
      createTestSpec();
      
      execSync(`node "${join(projectRoot, 'bin/spekk.js')}" show`, { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');
      
      // Verify document-level event listener is set up for delegation
      assert.ok(htmlContent.includes('document.addEventListener'), 'Should have document-level event listener for delegation');
      assert.ok(htmlContent.includes('click'), 'Event listener should handle click events');
      
    } finally {
      cleanupTestDir();
    }
  });
});