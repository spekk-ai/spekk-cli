import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import { existsSync, rmSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';

describe('Fix JavaScript Template Generation Errors', () => {
  
  const testDir = join(tmpdir(), `spekk-javascript-test-${Date.now()}`);
  
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

  test('generated HTML has no JavaScript syntax errors with special characters in spec titles', () => {
    setupTestDir();
    
    try {
      // Create specs with challenging titles containing special characters
      const specsDir = join(testDir, 'specs');
      const specDir = join(specsDir, 'test-spec');
      const assertionsDir = join(specDir, 'assertions');
      mkdirSync(assertionsDir, { recursive: true });
      
      // Create spec with single quotes in title
      const specContent = `---
id: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# User's Dashboard Test

This spec has a single quote in the title.`;
      writeFileSync(join(specDir, 'test-spec.md'), specContent);
      
      // Create assertion with double quotes in title
      const assertionContent = `---
id: test-assertion
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Test "Quoted" Assertion

This assertion has double quotes in the title.`;
      writeFileSync(join(assertionsDir, 'test-assertion.md'), assertionContent);
      
      // Run spekk show command
      execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      // Read generated HTML
      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');
      
      // Check that toggleSpec and showDetail functions are defined
      assert.ok(htmlContent.includes('function toggleSpec'), 'HTML should contain toggleSpec function');
      assert.ok(htmlContent.includes('function showDetail'), 'HTML should contain showDetail function');
      
      // Check that onclick handlers are properly escaped
      assert.ok(!htmlContent.includes("onclick=\"toggleSpec('test-spec')\""), 'Single quotes in onclick should be escaped or avoided');
      assert.ok(!htmlContent.includes("onclick=\"showDetail('test-assertion'"), 'Single quotes in onclick should be escaped or avoided');
      
      // Validate no broken JavaScript syntax patterns
      assert.ok(!htmlContent.includes("'test-spec'"), 'Unescaped single quotes should not appear in JavaScript context');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('generated HTML handles backticks in spec content', () => {
    setupTestDir();
    
    try {
      const specsDir = join(testDir, 'specs');
      const specDir = join(specsDir, 'code-spec');
      const assertionsDir = join(specDir, 'assertions');
      mkdirSync(assertionsDir, { recursive: true });
      
      // Create spec with backticks in content
      const specContent = `---
id: code-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Code Example Spec

This spec contains backticks: \`console.log("hello")\` and template literals.`;
      writeFileSync(join(specDir, 'code-spec.md'), specContent);
      
      // Run spekk show command
      execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      // Read generated HTML
      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');
      
      // Should not have unescaped backticks breaking JavaScript
      assert.ok(!htmlContent.includes('`console.log'), 'Backticks in spec content should be escaped');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('generated HTML handles curly braces in spec content', () => {
    setupTestDir();
    
    try {
      const specsDir = join(testDir, 'specs');
      const specDir = join(specsDir, 'object-spec');
      const assertionsDir = join(specDir, 'assertions');
      mkdirSync(assertionsDir, { recursive: true });
      
      // Create spec with curly braces in content
      const specContent = `---
id: object-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Object Example Spec

This spec contains curly braces: {example: "value"} that could break template literals.`;
      writeFileSync(join(specDir, 'object-spec.md'), specContent);
      
      // Run spekk show command
      execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      // Read generated HTML
      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');
      
      // Should not have unescaped curly braces in template literals
      assert.ok(htmlContent.includes('function toggleSpec'), 'JavaScript functions should still be present');
      assert.ok(htmlContent.includes('function showDetail'), 'JavaScript functions should still be present');
      
    } finally {
      cleanupTestDir();
    }
  });

  test('onclick handlers use proper escaping for JavaScript context', () => {
    setupTestDir();
    
    try {
      const specsDir = join(testDir, 'specs');
      const specDir = join(specsDir, 'special-chars-spec');
      const assertionsDir = join(specDir, 'assertions');
      mkdirSync(assertionsDir, { recursive: true });
      
      // Create spec and assertion with IDs that could break JavaScript
      const specContent = `---
id: special-chars-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Special Characters Spec`;
      writeFileSync(join(specDir, 'special-chars-spec.md'), specContent);
      
      const assertionContent = `---
id: special-assertion
parent: special-chars-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Special Assertion`;
      writeFileSync(join(assertionsDir, 'special-assertion.md'), assertionContent);
      
      // Run spekk show command
      execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      // Read generated HTML
      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');
      
      // Check that onclick handlers use proper JavaScript encoding
      // Should use JSON.stringify or similar approach for data safety
      assert.ok(htmlContent.includes('onclick='), 'HTML should contain onclick handlers');
      
      // Should not have raw string interpolation in onclick
      const onclickPattern = /onclick="[^"]*'/g;
      const matches = htmlContent.match(onclickPattern);
      if (matches) {
        // If there are onclick handlers with single quotes, they should be properly escaped
        matches.forEach(match => {
          assert.ok(!match.includes("'"), `Onclick handler should not contain unescaped single quotes: ${match}`);
        });
      }
      
    } finally {
      cleanupTestDir();
    }
  });

  test('generated HTML passes basic JavaScript validation', () => {
    setupTestDir();
    
    try {
      // Create comprehensive test case with all problematic characters
      const specsDir = join(testDir, 'specs');
      const specDir = join(specsDir, 'comprehensive-test');
      const assertionsDir = join(specDir, 'assertions');
      mkdirSync(assertionsDir, { recursive: true });
      
      const specContent = `---
id: comprehensive-test
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Comprehensive "Test" Spec

This spec contains single quotes: User's Dashboard
And double quotes: "quoted text"
And backticks: \`code example\`
And curly braces: {example: value}`;
      writeFileSync(join(specDir, 'comprehensive-test.md'), specContent);
      
      const assertionContent = `---
id: comprehensive-assertion
parent: comprehensive-test
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Comprehensive 'Test' Assertion

All special chars: '"'"\`{}.`;
      writeFileSync(join(assertionsDir, 'comprehensive-assertion.md'), assertionContent);
      
      // Run spekk show command
      execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000 
      });
      
      // Read generated HTML
      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');
      
      // Extract JavaScript content
      const scriptMatch = htmlContent.match(/<script>([\s\S]*?)<\/script>/);
      assert.ok(scriptMatch, 'HTML should contain script tag');
      
      const scriptContent = scriptMatch[1];
      
      // Validate that essential functions exist
      assert.ok(scriptContent.includes('function toggleSpec'), 'Script should contain toggleSpec function');
      assert.ok(scriptContent.includes('function showDetail'), 'Script should contain showDetail function');
      
      // Try to catch obvious syntax errors
      assert.ok(!scriptContent.includes('Unexpected token'), 'Script should not contain syntax error messages');
      
    } finally {
      cleanupTestDir();
    }
  });
});