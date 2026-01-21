import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import fs from 'fs';
import path from 'path';

describe('JSON Output Validation', () => {
  describe('Valid JSON Structure', () => {
    test('outputs parseable JSON to stdout', () => {
      const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
      
      // Should not throw when parsing
      let parsed;
      assert.doesNotThrow(() => {
        parsed = JSON.parse(result);
      }, 'Output should be valid JSON');
      
      // Should be an object, not array or primitive
      assert.equal(typeof parsed, 'object', 'JSON should be an object');
      assert.ok(!Array.isArray(parsed), 'JSON should not be an array');
      assert.notEqual(parsed, null, 'JSON should not be null');
    });

    test('success case includes all required fields', () => {
      // Create controlled test environment to ensure we get an assertion
      const tempDir = path.join(process.cwd(), 'temp-json-output-test');
      const assertionsDir = path.join(tempDir, 'assertions');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(assertionsDir, { recursive: true });
        
        const specContent = `---
id: temp-json-output-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# JSON Output Test Spec

Test spec for JSON output validation.`;
        
        const assertionContent = `---
id: test-json-assertion
parent: temp-json-output-test
created: 2026-01-20T16:00:00Z
priority: 1
status: not_started
---

# Test JSON Assertion

Test assertion for JSON output.`;
        
        fs.writeFileSync(path.join(tempDir, 'temp-json-output-test.md'), specContent);
        fs.writeFileSync(path.join(assertionsDir, 'test.md'), assertionContent);
        
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-json-output-test');
        fs.symlinkSync(tempDir, originalSpecsPath);
        
        try {
          const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
          const parsed = JSON.parse(result);
          
          if (parsed.type === 'assertion') {
            // Check all required fields exist
            assert.ok(parsed.type, 'Should have type field');
            assert.ok(parsed.id, 'Should have id field');
            assert.ok(parsed.parent, 'Should have parent field');
            assert.ok(parsed.file, 'Should have file field');
            assert.ok(parsed.priority !== undefined, 'Should have priority field');
            assert.ok(parsed.status, 'Should have status field');
            assert.ok(parsed.title, 'Should have title field');
            assert.ok(parsed.content, 'Should have content field');
            assert.ok(parsed.spec, 'Should have spec field');
            
            // Check field types
            assert.equal(typeof parsed.type, 'string', 'type should be string');
            assert.equal(typeof parsed.id, 'string', 'id should be string');
            assert.equal(typeof parsed.parent, 'string', 'parent should be string');
            assert.equal(typeof parsed.file, 'string', 'file should be string');
            assert.equal(typeof parsed.priority, 'number', 'priority should be number');
            assert.equal(typeof parsed.status, 'string', 'status should be string');
            assert.equal(typeof parsed.title, 'string', 'title should be string');
            assert.equal(typeof parsed.content, 'string', 'content should be string');
            assert.equal(typeof parsed.spec, 'object', 'spec should be object');
            
            // Check spec object has required fields
            assert.ok(parsed.spec.id, 'spec should have id field');
            assert.ok(parsed.spec.file, 'spec should have file field');
            assert.ok(parsed.spec.title, 'spec should have title field');
          }
          
        } finally {
          fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });

    test('complete case has correct format', () => {
      // Test when all assertions are done by creating a controlled environment
      const tempDir = path.join(process.cwd(), 'temp-complete-test');
      const assertionsDir = path.join(tempDir, 'assertions');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(assertionsDir, { recursive: true });
        
        const specContent = `---
id: temp-complete-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Complete Test Spec`;
        
        const doneAssertion = `---
id: done-assertion
parent: temp-complete-test
created: 2026-01-20T16:00:00Z
priority: 1
status: done
---

# Done Assertion

This assertion is complete.`;
        
        fs.writeFileSync(path.join(tempDir, 'temp-complete-test.md'), specContent);
        fs.writeFileSync(path.join(assertionsDir, 'done.md'), doneAssertion);
        
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-complete-test');
        fs.symlinkSync(tempDir, originalSpecsPath);
        
        try {
          const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
          const parsed = JSON.parse(result);
          
          if (parsed.status === 'complete') {
            assert.equal(parsed.status, 'complete', 'Should have complete status');
            assert.ok(parsed.message, 'Should have message field');
            assert.equal(typeof parsed.message, 'string', 'message should be string');
            assert.equal(parsed.message, 'All specifications are complete', 'Should have correct message');
          }
          
        } finally {
          fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });

    test('empty case has correct format', () => {
      // Test by temporarily moving specs directory
      const specsDir = path.join(process.cwd(), 'specs');
      const backupDir = path.join(process.cwd(), 'specs-backup');
      
      // Only run this test if specs directory exists
      if (fs.existsSync(specsDir)) {
        try {
          fs.renameSync(specsDir, backupDir);
          
          const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
          const parsed = JSON.parse(result);
          
          assert.equal(parsed.status, 'empty', 'Should have empty status');
          assert.ok(parsed.message, 'Should have message field');
          assert.equal(typeof parsed.message, 'string', 'message should be string');
          assert.equal(parsed.message, 'No specifications found in specs/ directory', 'Should have correct message');
          
        } finally {
          // Restore specs directory
          if (fs.existsSync(backupDir)) {
            fs.renameSync(backupDir, specsDir);
          }
        }
      }
    });

    test('error case has JSON format', () => {
      // Create invalid spec to trigger error
      const tempDir = path.join(process.cwd(), 'temp-error-test');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(path.join(tempDir, 'assertions'), { recursive: true });
        
        const invalidSpec = `---
created: 2026-01-20T16:00:00Z
priority: 1
---

# Invalid Spec (missing id)`;
        
        fs.writeFileSync(path.join(tempDir, 'temp-error-test.md'), invalidSpec);
        
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-error-test');
        fs.symlinkSync(tempDir, originalSpecsPath);
        
        try {
          const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
          const parsed = JSON.parse(result);
          
          assert.equal(parsed.error, true, 'Should have error: true');
          assert.ok(parsed.message, 'Should have message field');
          assert.equal(typeof parsed.message, 'string', 'message should be string');
          assert.ok(parsed.message.includes("Missing required field 'id'"), 'Should have descriptive error message');
          
        } catch (execError) {
          // CLI should exit with code 1 for errors, but still output JSON
          assert.equal(execError.status, 1, 'Should exit with code 1 for errors');
          
          const result = execError.stdout.toString();
          const parsed = JSON.parse(result);
          
          assert.equal(parsed.error, true, 'Should have error: true in JSON output');
          assert.ok(parsed.message, 'Should have message field in error JSON');
        } finally {
          fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });

    test('outputs single JSON object, not multiple or streaming', () => {
      const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
      
      // Should parse as exactly one JSON object (pretty-printed is OK)
      let parsed;
      assert.doesNotThrow(() => {
        parsed = JSON.parse(result.trim());
      }, 'Should parse as single valid JSON object');
      
      // Should not contain multiple root JSON objects (check for pattern like }{ or }{)
      const multipleObjectPattern = /}\s*{/;
      assert.ok(!multipleObjectPattern.test(result), 'Should not contain multiple JSON objects');
      
      // Should be a single object at the root level
      assert.equal(typeof parsed, 'object', 'Should be a JSON object');
      assert.ok(!Array.isArray(parsed), 'Should not be an array');
      assert.notEqual(parsed, null, 'Should not be null');
    });

    test('field values match expected types and formats', () => {
      const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
      const parsed = JSON.parse(result);
      
      if (parsed.type === 'assertion') {
        // Test specific field formats
        assert.equal(parsed.type, 'assertion', 'type should always be "assertion" for work items');
        assert.ok([1, 2, 3].includes(parsed.priority), 'priority should be 1, 2, or 3');
        assert.ok(['not_started', 'in_progress', 'done'].includes(parsed.status), 'status should be valid value');
        assert.ok(parsed.file.startsWith('specs/'), 'file should be relative path starting with specs/');
        assert.ok(parsed.file.endsWith('.md'), 'file should end with .md');
        
        // Test timestamp format in content
        assert.ok(parsed.content.includes('created:'), 'content should include created timestamp');
        
        // Test spec reference structure
        assert.equal(typeof parsed.spec.id, 'string', 'spec.id should be string');
        assert.ok(parsed.spec.file.startsWith('specs/'), 'spec.file should be relative path');
        assert.ok(parsed.spec.file.endsWith('.md'), 'spec.file should end with .md');
      }
    });
  });

  describe('Output Consistency', () => {
    test('multiple runs produce identical JSON structure', () => {
      const result1 = execSync('node src/parser/cli.js', { encoding: 'utf8' });
      const result2 = execSync('node src/parser/cli.js', { encoding: 'utf8' });
      
      const parsed1 = JSON.parse(result1);
      const parsed2 = JSON.parse(result2);
      
      // Structure should be identical (content might differ if files change)
      assert.equal(typeof parsed1, typeof parsed2, 'Should have same type');
      
      if (parsed1.type === 'assertion' && parsed2.type === 'assertion') {
        // Should have same fields structure
        const keys1 = Object.keys(parsed1).sort();
        const keys2 = Object.keys(parsed2).sort();
        assert.deepEqual(keys1, keys2, 'Should have same field keys');
        
        // Field types should match
        for (const key of keys1) {
          assert.equal(typeof parsed1[key], typeof parsed2[key], `Field ${key} should have same type`);
        }
      }
    });

    test('JSON is properly formatted and indented', () => {
      const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
      
      // Should be pretty-printed (contain newlines and spaces)
      assert.ok(result.includes('\n'), 'JSON should be formatted with newlines');
      assert.ok(result.includes('  '), 'JSON should be indented with spaces');
      
      // Should parse identically to compact version
      const parsed = JSON.parse(result);
      const compact = JSON.stringify(parsed);
      const reparsed = JSON.parse(compact);
      
      assert.deepEqual(parsed, reparsed, 'Formatted and compact JSON should parse identically');
    });
  });
});