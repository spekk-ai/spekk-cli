import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import { existsSync, rmSync, mkdirSync, writeFileSync, readFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { tmpdir } from 'node:os';

describe('Web Interface Detail Badges Format', () => {
  
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
status: in_progress
---

# Test Spec

This is a test specification for testing badges.`;
    writeFileSync(join(specDir, 'test-spec.md'), specContent);
    
    const assertionContent = `---
id: test-assertion
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 2
status: done
---

# Test Assertion

This is a test assertion for testing badges.`;
    writeFileSync(join(assertionsDir, 'test-assertion.md'), assertionContent);
  }

  test('status badge shows only icon (no text)', () => {
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
      
      // Verify status badges contain only icons, no text labels
      assert.ok(
        htmlContent.includes('<span class="detail-status-badge">🔄</span>') ||
        htmlContent.includes('<span class="detail-status-badge">✅</span>'), 
        'Status badges should contain only icons without text labels'
      );
      
      // Verify no text labels in status badges
      assert.ok(!htmlContent.includes('>done<'), 'Status badge should not contain "done" text');
      assert.ok(!htmlContent.includes('>in_progress<'), 'Status badge should not contain "in_progress" text');
      assert.ok(!htmlContent.includes('>not_started<'), 'Status badge should not contain "not_started" text');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('priority badge shows only number (no emoji)', () => {
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
      
      // Verify priority badges contain only numbers
      assert.ok(
        htmlContent.includes('<span class="detail-priority-badge">1</span>') ||
        htmlContent.includes('<span class="detail-priority-badge">2</span>') ||
        htmlContent.includes('<span class="detail-priority-badge">3</span>'), 
        'Priority badges should contain only numbers without emoji decorations'
      );
      
      // Verify no emoji decorations in priority badges
      assert.ok(!htmlContent.includes('🔥'), 'Priority badge should not contain fire emoji');
      assert.ok(!htmlContent.includes('⚠️'), 'Priority badge should not contain warning emoji');  
      assert.ok(!htmlContent.includes('💡'), 'Priority badge should not contain lightbulb emoji');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('functions return simplified HTML markup', () => {
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
      
      // Verify simplified status badge format (no status-specific classes)
      assert.ok(
        !htmlContent.includes('class="detail-status-badge status-'),
        'Status badges should not include status-specific CSS classes'
      );
      
      // Verify detail meta shows correct format
      assert.ok(htmlContent.includes('Status:'), 'Detail meta should show "Status:" label');
      assert.ok(htmlContent.includes('Priority:'), 'Detail meta should show "Priority:" label');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('detail view matches tree view formatting style', () => {
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
      
      // Verify consistency between tree view and detail view
      // Both should use simple number/icon format
      assert.ok(
        htmlContent.includes('class="priority-badge') && 
        htmlContent.includes('class="detail-priority-badge'),
        'Both tree view and detail view should have priority badges'
      );
      
      assert.ok(
        htmlContent.includes('class="status-badge') && 
        htmlContent.includes('class="detail-status-badge'),
        'Both tree view and detail view should have status badges'
      );
      
    } finally {
      cleanupTestDir();
    }
  });
});