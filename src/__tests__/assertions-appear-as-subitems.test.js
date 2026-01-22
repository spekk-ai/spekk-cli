import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import { existsSync, rmSync, mkdirSync, writeFileSync, readFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { tmpdir } from 'node:os';

describe('Assertions Appear as Sub-items', () => {
  
  const testDir = join(tmpdir(), `spekk-test-assertions-${Date.now()}`);
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

  test('assertions appear as sub-items under parent specs in tree structure', () => {
    setupTestDir();
    
    try {
      // Create test specs directory structure
      const specsDir = join(testDir, 'specs');
      const spec1Dir = join(specsDir, 'test-spec-1');
      const spec2Dir = join(specsDir, 'test-spec-2');
      const assertions1Dir = join(spec1Dir, 'assertions');
      const assertions2Dir = join(spec2Dir, 'assertions');
      
      mkdirSync(assertions1Dir, { recursive: true });
      mkdirSync(assertions2Dir, { recursive: true });
      
      // Create first spec with two assertions
      const spec1Content = `---
id: test-spec-1
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Test Spec 1

This is the first test specification.`;
      writeFileSync(join(spec1Dir, 'test-spec-1.md'), spec1Content);
      
      const assertion1Content = `---
id: test-assertion-1
parent: test-spec-1
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Test Assertion 1

This is the first test assertion.`;
      writeFileSync(join(assertions1Dir, 'test-assertion-1.md'), assertion1Content);
      
      const assertion2Content = `---
id: test-assertion-2
parent: test-spec-1
created: 2026-01-22T21:01:00Z
priority: 2
status: in_progress
---

# Test Assertion 2

This is the second test assertion.`;
      writeFileSync(join(assertions1Dir, 'test-assertion-2.md'), assertion2Content);
      
      // Create second spec with one assertion
      const spec2Content = `---
id: test-spec-2
created: 2026-01-22T22:00:00Z
priority: 2
status: not_started
---

# Test Spec 2

This is the second test specification.`;
      writeFileSync(join(spec2Dir, 'test-spec-2.md'), spec2Content);
      
      const assertion3Content = `---
id: test-assertion-3
parent: test-spec-2
created: 2026-01-22T22:00:00Z
priority: 1
status: done
---

# Test Assertion 3

This is the third test assertion.`;
      writeFileSync(join(assertions2Dir, 'test-assertion-3.md'), assertion3Content);
      
      // Run spekk show command
      const spekkBin = join(projectRoot, 'bin/spekk.js');
      execSync(`node "${spekkBin}" show`, { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      // Read generated HTML
      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');
      
      // Verify assertions are nested under correct parent specs
      
      // Check that spec 1 has its assertions nested
      assert.ok(htmlContent.includes('id="assertions-test-spec-1"'), 'Should have assertions container for test-spec-1');
      assert.ok(htmlContent.includes('test-assertion-1'), 'Should include test-assertion-1');
      assert.ok(htmlContent.includes('test-assertion-2'), 'Should include test-assertion-2');
      
      // Check that spec 2 has its assertion nested  
      assert.ok(htmlContent.includes('id="assertions-test-spec-2"'), 'Should have assertions container for test-spec-2');
      assert.ok(htmlContent.includes('test-assertion-3'), 'Should include test-assertion-3');
      
      // Verify CSS classes for visual hierarchy
      assert.ok(htmlContent.includes('class="assertions-list"'), 'Should use assertions-list class for sub-items');
      assert.ok(htmlContent.includes('class="assertion-item"'), 'Should use assertion-item class for individual assertions');
      
      // Verify assertions are indented (check for margin-left styling)
      assert.ok(htmlContent.includes('margin-left: 20px'), 'Assertions should be indented with margin-left');
      
      // Verify assertions show status badges
      assert.ok(htmlContent.includes('status-not_started'), 'Should show not_started status');
      assert.ok(htmlContent.includes('status-in_progress'), 'Should show in_progress status');
      assert.ok(htmlContent.includes('status-done'), 'Should show done status');
      
      // Verify assertions show priority badges
      assert.ok(htmlContent.includes('priority-1'), 'Should show priority 1');
      assert.ok(htmlContent.includes('priority-2'), 'Should show priority 2');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('assertions are visually distinguished from specs with proper styling', () => {
    setupTestDir();
    
    try {
      // Create minimal test structure
      const specsDir = join(testDir, 'specs');
      const specDir = join(specsDir, 'visual-test-spec');
      const assertionsDir = join(specDir, 'assertions');
      mkdirSync(assertionsDir, { recursive: true });
      
      const specContent = `---
id: visual-test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Visual Test Spec`;
      writeFileSync(join(specDir, 'visual-test-spec.md'), specContent);
      
      const assertionContent = `---
id: visual-assertion
parent: visual-test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Visual Assertion`;
      writeFileSync(join(assertionsDir, 'visual-assertion.md'), assertionContent);
      
      // Run spekk show command
      const spekkBin = join(projectRoot, 'bin/spekk.js');
      execSync(`node "${spekkBin}" show`, { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');
      
      // Verify different visual styling between specs and assertions
      assert.ok(htmlContent.includes('class="spec-header"'), 'Specs should use spec-header class');
      assert.ok(htmlContent.includes('class="assertion-item"'), 'Assertions should use assertion-item class');
      
      // Check for different background colors
      assert.ok(htmlContent.includes('background: #f1f5f9'), 'Specs should have gray background');
      assert.ok(htmlContent.includes('background: white'), 'Assertions should have white background');
      
      // Check for border styling on assertions
      assert.ok(htmlContent.includes('border-left: 3px solid'), 'Assertions should have left border');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('empty specs can be expanded but show appropriate structure', () => {
    setupTestDir();
    
    try {
      // Create spec with no assertions
      const specsDir = join(testDir, 'specs');
      const specDir = join(specsDir, 'empty-spec');
      const assertionsDir = join(specDir, 'assertions');
      mkdirSync(assertionsDir, { recursive: true });
      
      const specContent = `---
id: empty-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Empty Spec

This spec has no assertions yet.`;
      writeFileSync(join(specDir, 'empty-spec.md'), specContent);
      
      // Run spekk show command
      const spekkBin = join(projectRoot, 'bin/spekk.js');
      execSync(`node "${spekkBin}" show`, { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');
      
      // Verify empty spec structure exists but is empty
      assert.ok(htmlContent.includes('id="assertions-empty-spec"'), 'Should have assertions container for empty spec');
      assert.ok(htmlContent.includes('class="toggle-icon"'), 'Empty specs should still be expandable');
      
      // The assertions list should be empty but still have proper structure
      const assertionsMatch = htmlContent.match(/id="assertions-empty-spec"[^>]*>(.*?)<\/ul>/s);
      assert.ok(assertionsMatch, 'Should find assertions container');
      const assertionsContent = assertionsMatch[1].trim();
      
      // Should be empty or contain only whitespace
      assert.ok(assertionsContent.length === 0 || /^\s*$/.test(assertionsContent), 'Empty spec should have empty assertions list');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('assertions are organized under correct parent spec', () => {
    setupTestDir();
    
    try {
      // Create multiple specs to test correct grouping
      const specsDir = join(testDir, 'specs');
      
      // Create spec A with assertions 1,2
      const specADir = join(specsDir, 'spec-a');
      const assertionsADir = join(specADir, 'assertions');
      mkdirSync(assertionsADir, { recursive: true });
      
      writeFileSync(join(specADir, 'spec-a.md'), `---
id: spec-a
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Spec A`);
      
      writeFileSync(join(assertionsADir, 'assertion-a1.md'), `---
id: assertion-a1
parent: spec-a
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Assertion A1`);
      
      writeFileSync(join(assertionsADir, 'assertion-a2.md'), `---
id: assertion-a2
parent: spec-a
created: 2026-01-22T21:01:00Z
priority: 2
status: not_started
---

# Assertion A2`);
      
      // Create spec B with assertion 3
      const specBDir = join(specsDir, 'spec-b');
      const assertionsBDir = join(specBDir, 'assertions');
      mkdirSync(assertionsBDir, { recursive: true });
      
      writeFileSync(join(specBDir, 'spec-b.md'), `---
id: spec-b
created: 2026-01-22T22:00:00Z
priority: 2
status: not_started
---

# Spec B`);
      
      writeFileSync(join(assertionsBDir, 'assertion-b1.md'), `---
id: assertion-b1
parent: spec-b
created: 2026-01-22T22:00:00Z
priority: 1
status: not_started
---

# Assertion B1`);
      
      // Run spekk show command
      const spekkBin = join(projectRoot, 'bin/spekk.js');
      execSync(`node "${spekkBin}" show`, { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');
      
      // Extract assertions for spec A
      const specAMatch = htmlContent.match(/id="assertions-spec-a"[^>]*>(.*?)(?=<\/ul>)/s);
      assert.ok(specAMatch, 'Should find assertions container for spec-a');
      const specAAssertions = specAMatch[1];
      
      // Extract assertions for spec B  
      const specBMatch = htmlContent.match(/id="assertions-spec-b"[^>]*>(.*?)(?=<\/ul>)/s);
      assert.ok(specBMatch, 'Should find assertions container for spec-b');
      const specBAssertions = specBMatch[1];
      
      // Verify correct grouping
      assert.ok(specAAssertions.includes('assertion-a1'), 'Spec A should contain assertion-a1');
      assert.ok(specAAssertions.includes('assertion-a2'), 'Spec A should contain assertion-a2');
      assert.ok(!specAAssertions.includes('assertion-b1'), 'Spec A should NOT contain assertion-b1');
      
      assert.ok(specBAssertions.includes('assertion-b1'), 'Spec B should contain assertion-b1');
      assert.ok(!specBAssertions.includes('assertion-a1'), 'Spec B should NOT contain assertion-a1');
      assert.ok(!specBAssertions.includes('assertion-a2'), 'Spec B should NOT contain assertion-a2');
      
    } finally {
      cleanupTestDir();
    }
  });
});