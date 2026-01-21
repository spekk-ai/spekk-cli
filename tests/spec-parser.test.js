import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import fs from 'fs';
import path from 'path';
import { parseAllSpecs, parseFrontmatter, validateFields, findNextAssertion } from '../src/parser/index.js';

describe('Spec Parser Field Validation', () => {
  describe('Required Fields Validation', () => {
    test('validates required fields for specs', () => {
      const tempDir = path.join(process.cwd(), 'temp-spec-required-fields');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(path.join(tempDir, 'assertions'), { recursive: true });
        
        // Test missing ID
        const missingIdSpec = `---
created: 2026-01-20T16:00:00Z
priority: 1
---

# Spec Without ID

This spec is missing the id field.`;
        
        fs.writeFileSync(path.join(tempDir, 'temp-spec-required-fields.md'), missingIdSpec);
        
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-spec-required-fields');
        fs.symlinkSync(tempDir, originalSpecsPath);
        
        try {
          parseAllSpecs();
          assert.fail('Should have thrown error for missing id field');
        } catch (error) {
          assert.ok(error.message.includes("Missing required field 'id'"), 
            'Should detect missing id field in spec');
          assert.ok(error.message.includes('temp-spec-required-fields.md'), 
            'Error should include file path');
        } finally {
          fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });

    test('validates required fields for assertions', () => {
      const tempDir = path.join(process.cwd(), 'temp-assertion-required-fields');
      const assertionsDir = path.join(tempDir, 'assertions');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(assertionsDir, { recursive: true });
        
        // Create valid spec first
        const specContent = `---
id: temp-assertion-required-fields
created: 2026-01-20T16:00:00Z
priority: 1
---

# Test Spec for Assertion Validation

This spec tests assertion field validation.`;
        
        fs.writeFileSync(path.join(tempDir, 'temp-assertion-required-fields.md'), specContent);
        
        // Test missing parent field
        const missingParentAssertion = `---
id: test-assertion
created: 2026-01-20T16:00:00Z
priority: 1
---

# Assertion Without Parent

This assertion is missing the parent field.`;
        
        fs.writeFileSync(path.join(assertionsDir, 'missing-parent.md'), missingParentAssertion);
        
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-assertion-required-fields');
        fs.symlinkSync(tempDir, originalSpecsPath);
        
        try {
          parseAllSpecs();
          assert.fail('Should have thrown error for missing parent field');
        } catch (error) {
          assert.ok(error.message.includes("Missing required field 'parent'"), 
            'Should detect missing parent field in assertion');
          assert.ok(error.message.includes('missing-parent.md'), 
            'Error should include assertion file name');
        } finally {
          fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });
  });

  describe('Field Format Validation', () => {
    test('validates kebab-case ID format', () => {
      const tempDir = path.join(process.cwd(), 'temp-invalid-id-format');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(path.join(tempDir, 'assertions'), { recursive: true });
        
        const invalidIds = [
          'CamelCase',           // should be lowercase
          'snake_case',          // should use hyphens
          'with spaces',         // no spaces allowed
          'with@special!chars',  // no special chars
          'double--hyphens',     // consecutive hyphens
          '-leading-hyphen',     // can't start with hyphen
          'trailing-hyphen-',    // can't end with hyphen
          '123-numeric-start'    // numbers are ok but this tests pattern
        ];
        
        for (let i = 0; i < invalidIds.length; i++) {
          const invalidId = invalidIds[i];
          
          const specContent = `---
id: ${invalidId}
created: 2026-01-20T16:00:00Z
priority: 1
---

# Invalid ID Test

This spec has an invalid ID format.`;
          
          fs.writeFileSync(path.join(tempDir, 'temp-invalid-id-format.md'), specContent);
          
          const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-invalid-id-format');
          if (!fs.existsSync(originalSpecsPath)) {
            fs.symlinkSync(tempDir, originalSpecsPath);
          }
          
          try {
            parseAllSpecs();
            assert.fail(`Should have thrown error for invalid ID format: ${invalidId}`);
          } catch (error) {
            assert.ok(error.message.includes(`Invalid id format '${invalidId}'`), 
              `Should detect invalid ID format: ${invalidId}`);
            assert.ok(error.message.includes('kebab-case'), 
              'Error should mention kebab-case requirement');
          } finally {
            if (fs.existsSync(originalSpecsPath)) {
              fs.unlinkSync(originalSpecsPath);
            }
          }
          
          // Clean up for next iteration
          fs.unlinkSync(path.join(tempDir, 'temp-invalid-id-format.md'));
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });

    test('validates ISO 8601 timestamp format', () => {
      const tempDir = path.join(process.cwd(), 'temp-invalid-timestamp');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(path.join(tempDir, 'assertions'), { recursive: true });
        
        const invalidTimestamps = [
          '2026-01-20',              // missing time
          '2026-01-20T16:00:00',     // missing Z
          '2026-1-20T16:00:00Z',     // wrong month format
          '2026-01-20 16:00:00Z',    // space instead of T
          '01/20/2026T16:00:00Z',    // US date format
          'invalid-timestamp'         // completely invalid
        ];
        
        for (let i = 0; i < invalidTimestamps.length; i++) {
          const invalidTimestamp = invalidTimestamps[i];
          
          const specContent = `---
id: temp-invalid-timestamp
created: ${invalidTimestamp}
priority: 1
---

# Invalid Timestamp Test

This spec has an invalid timestamp format.`;
          
          fs.writeFileSync(path.join(tempDir, 'temp-invalid-timestamp.md'), specContent);
          
          const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-invalid-timestamp');
          if (!fs.existsSync(originalSpecsPath)) {
            fs.symlinkSync(tempDir, originalSpecsPath);
          }
          
          try {
            parseAllSpecs();
            assert.fail(`Should have thrown error for invalid timestamp: ${invalidTimestamp}`);
          } catch (error) {
            assert.ok(error.message.includes('Invalid ISO 8601 timestamp'), 
              `Should detect invalid timestamp format: ${invalidTimestamp}`);
            assert.ok(error.message.includes(invalidTimestamp), 
              `Error should mention the invalid timestamp: ${invalidTimestamp}`);
          } finally {
            if (fs.existsSync(originalSpecsPath)) {
              fs.unlinkSync(originalSpecsPath);
            }
          }
          
          // Clean up for next iteration
          fs.unlinkSync(path.join(tempDir, 'temp-invalid-timestamp.md'));
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });

    test('validates priority values must be 1, 2, or 3', () => {
      const tempDir = path.join(process.cwd(), 'temp-invalid-priority');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(path.join(tempDir, 'assertions'), { recursive: true });
        
        const invalidPriorities = [0, 4, 5, -1, 10, 'high', 'medium', 'low'];
        
        for (let i = 0; i < invalidPriorities.length; i++) {
          const invalidPriority = invalidPriorities[i];
          
          const specContent = `---
id: temp-invalid-priority
created: 2026-01-20T16:00:00Z
priority: ${invalidPriority}
---

# Invalid Priority Test

This spec has an invalid priority value.`;
          
          fs.writeFileSync(path.join(tempDir, 'temp-invalid-priority.md'), specContent);
          
          const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-invalid-priority');
          if (!fs.existsSync(originalSpecsPath)) {
            fs.symlinkSync(tempDir, originalSpecsPath);
          }
          
          try {
            parseAllSpecs();
            assert.fail(`Should have thrown error for invalid priority: ${invalidPriority}`);
          } catch (error) {
            assert.ok(error.message.includes(`Invalid priority value '${invalidPriority}'`), 
              `Should detect invalid priority: ${invalidPriority}`);
            assert.ok(error.message.includes('must be: 1, 2, or 3'), 
              'Error should specify valid priority values');
          } finally {
            if (fs.existsSync(originalSpecsPath)) {
              fs.unlinkSync(originalSpecsPath);
            }
          }
          
          // Clean up for next iteration
          fs.unlinkSync(path.join(tempDir, 'temp-invalid-priority.md'));
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });
  });

  describe('Status Field Validation', () => {
    test('accepts valid status values', () => {
      const tempDir = path.join(process.cwd(), 'temp-valid-status');
      const assertionsDir = path.join(tempDir, 'assertions');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(assertionsDir, { recursive: true });
        
        const specContent = `---
id: temp-valid-status
created: 2026-01-20T16:00:00Z
priority: 1
---

# Valid Status Test Spec

Test spec for valid status values.`;
        
        fs.writeFileSync(path.join(tempDir, 'temp-valid-status.md'), specContent);
        
        const validStatuses = ['not_started', 'in_progress', 'done'];
        
        for (let i = 0; i < validStatuses.length; i++) {
          const status = validStatuses[i];
          const assertionContent = `---
id: test-${i}
parent: temp-valid-status
created: 2026-01-20T16:00:00Z
priority: 1
status: ${status}
---

# Test Valid Status ${i}

This tests status: ${status}`;
          
          fs.writeFileSync(path.join(assertionsDir, `test-${i}.md`), assertionContent);
        }
        
        // Test missing status (should default to not_started)
        const noStatusAssertion = `---
id: test-no-status
parent: temp-valid-status
created: 2026-01-20T16:00:00Z
priority: 1
---

# Test No Status

This tests missing status field.`;
        
        fs.writeFileSync(path.join(assertionsDir, 'test-no-status.md'), noStatusAssertion);
        
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-valid-status');
        fs.symlinkSync(tempDir, originalSpecsPath);
        
        try {
          const { assertions } = parseAllSpecs();
          const testAssertions = assertions.filter(a => a.parent === 'temp-valid-status');
          
          // Should accept all valid statuses
          assert.ok(testAssertions.find(a => a.status === 'not_started'), 'Should accept not_started');
          assert.ok(testAssertions.find(a => a.status === 'in_progress'), 'Should accept in_progress');
          assert.ok(testAssertions.find(a => a.status === 'done'), 'Should accept done');
          
          // Should default missing status to not_started
          const noStatus = testAssertions.find(a => a.id === 'test-no-status');
          assert.equal(noStatus.status, 'not_started', 'Missing status should default to not_started');
          
        } finally {
          fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });

    test('rejects invalid status values', () => {
      const tempDir = path.join(process.cwd(), 'temp-invalid-status');
      const assertionsDir = path.join(tempDir, 'assertions');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(assertionsDir, { recursive: true });
        
        const specContent = `---
id: temp-invalid-status
created: 2026-01-20T16:00:00Z
priority: 1
---

# Invalid Status Test Spec

Test spec for invalid status validation.`;
        
        fs.writeFileSync(path.join(tempDir, 'temp-invalid-status.md'), specContent);
        
        const invalidStatuses = [
          'completed',      // should use 'done'
          'Not Started',    // wrong case/format
          'in-progress',    // wrong separator
          'todo',           // not valid
          'pending',        // not valid
          'finished'        // should use 'done'
        ];
        
        for (let i = 0; i < invalidStatuses.length; i++) {
          const invalidStatus = invalidStatuses[i];
          const assertionContent = `---
id: test-invalid-${i}
parent: temp-invalid-status
created: 2026-01-20T16:00:00Z
priority: 1
status: ${invalidStatus}
---

# Test Invalid Status ${i}

This tests invalid status: ${invalidStatus}`;
          
          fs.writeFileSync(path.join(assertionsDir, `test-invalid-${i}.md`), assertionContent);
          
          const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-invalid-status');
          if (!fs.existsSync(originalSpecsPath)) {
            fs.symlinkSync(tempDir, originalSpecsPath);
          }
          
          try {
            parseAllSpecs();
            assert.fail(`Should have thrown error for invalid status: ${invalidStatus}`);
          } catch (error) {
            assert.ok(error.message.includes(invalidStatus), 
              `Error should mention invalid status '${invalidStatus}'`);
            assert.ok(error.message.includes('not_started, in_progress, done'), 
              'Error should list valid status values');
          } finally {
            if (fs.existsSync(originalSpecsPath)) {
              fs.unlinkSync(originalSpecsPath);
            }
          }
          
          // Clean up for next iteration
          fs.unlinkSync(path.join(assertionsDir, `test-invalid-${i}.md`));
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });
  });

  describe('Duplicate ID Detection', () => {
    test('detects duplicate spec IDs across files', () => {
      const tempDir1 = path.join(process.cwd(), 'temp-duplicate-spec-1');
      const tempDir2 = path.join(process.cwd(), 'temp-duplicate-spec-2');
      
      try {
        fs.mkdirSync(tempDir1, { recursive: true });
        fs.mkdirSync(tempDir2, { recursive: true });
        fs.mkdirSync(path.join(tempDir1, 'assertions'), { recursive: true });
        fs.mkdirSync(path.join(tempDir2, 'assertions'), { recursive: true });
        
        const duplicateSpec1 = `---
id: duplicate-spec-id
created: 2026-01-20T16:00:00Z
priority: 1
---

# First Duplicate Spec

This is the first spec with duplicate ID.`;

        const duplicateSpec2 = `---
id: duplicate-spec-id
created: 2026-01-20T16:00:00Z
priority: 1
---

# Second Duplicate Spec

This is the second spec with duplicate ID.`;
        
        fs.writeFileSync(path.join(tempDir1, 'temp-duplicate-spec-1.md'), duplicateSpec1);
        fs.writeFileSync(path.join(tempDir2, 'temp-duplicate-spec-2.md'), duplicateSpec2);
        
        const originalSpecsPath1 = path.join(process.cwd(), 'specs', 'temp-duplicate-spec-1');
        const originalSpecsPath2 = path.join(process.cwd(), 'specs', 'temp-duplicate-spec-2');
        
        fs.symlinkSync(tempDir1, originalSpecsPath1);
        fs.symlinkSync(tempDir2, originalSpecsPath2);
        
        try {
          parseAllSpecs();
          assert.fail('Should have thrown error for duplicate spec IDs');
        } catch (error) {
          assert.ok(error.message.includes("Duplicate spec id 'duplicate-spec-id'"), 
            'Should detect duplicate spec IDs');
          assert.ok(error.message.includes('temp-duplicate-spec-1.md'), 
            'Error should include first file path');
          assert.ok(error.message.includes('temp-duplicate-spec-2.md'), 
            'Error should include second file path');
        } finally {
          fs.unlinkSync(originalSpecsPath1);
          fs.unlinkSync(originalSpecsPath2);
        }
        
      } finally {
        if (fs.existsSync(tempDir1)) fs.rmSync(tempDir1, { recursive: true });
        if (fs.existsSync(tempDir2)) fs.rmSync(tempDir2, { recursive: true });
      }
    });

    test('detects duplicate assertion IDs within spec', () => {
      const tempDir = path.join(process.cwd(), 'temp-duplicate-assertion');
      const assertionsDir = path.join(tempDir, 'assertions');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(assertionsDir, { recursive: true });
        
        const specContent = `---
id: temp-duplicate-assertion
created: 2026-01-20T16:00:00Z
priority: 1
---

# Duplicate Assertion Test Spec

Test spec for duplicate assertion IDs.`;
        
        fs.writeFileSync(path.join(tempDir, 'temp-duplicate-assertion.md'), specContent);
        
        const assertion1 = `---
id: duplicate-assertion-id
parent: temp-duplicate-assertion
created: 2026-01-20T16:00:00Z
priority: 1
---

# First Duplicate Assertion

First assertion with duplicate ID.`;

        const assertion2 = `---
id: duplicate-assertion-id
parent: temp-duplicate-assertion
created: 2026-01-20T16:00:00Z
priority: 1
---

# Second Duplicate Assertion

Second assertion with duplicate ID.`;
        
        fs.writeFileSync(path.join(assertionsDir, 'assertion-1.md'), assertion1);
        fs.writeFileSync(path.join(assertionsDir, 'assertion-2.md'), assertion2);
        
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-duplicate-assertion');
        fs.symlinkSync(tempDir, originalSpecsPath);
        
        try {
          parseAllSpecs();
          assert.fail('Should have thrown error for duplicate assertion IDs');
        } catch (error) {
          assert.ok(error.message.includes("Duplicate assertion id 'duplicate-assertion-id'"), 
            'Should detect duplicate assertion IDs');
          assert.ok(error.message.includes('temp-duplicate-assertion'), 
            'Error should mention the spec name');
          assert.ok(error.message.includes('assertion-1.md'), 
            'Error should include first assertion file');
          assert.ok(error.message.includes('assertion-2.md'), 
            'Error should include second assertion file');
        } finally {
          fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });
  });

  describe('Parent Reference Validation', () => {
    test('validates parent references exist for assertions', () => {
      const tempDir = path.join(process.cwd(), 'temp-parent-validation');
      const assertionsDir = path.join(tempDir, 'assertions');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(assertionsDir, { recursive: true });
        
        // Create spec file
        const specContent = `---
id: temp-parent-validation
created: 2026-01-20T16:00:00Z
priority: 1
---

# Parent Validation Test Spec

Test spec for parent reference validation.`;
        
        fs.writeFileSync(path.join(tempDir, 'temp-parent-validation.md'), specContent);
        
        // Create assertion with invalid parent reference
        const invalidParentAssertion = `---
id: test-invalid-parent
parent: nonexistent-spec
created: 2026-01-20T16:00:00Z
priority: 1
---

# Test Invalid Parent

This assertion references a nonexistent parent spec.`;
        
        fs.writeFileSync(path.join(assertionsDir, 'invalid-parent.md'), invalidParentAssertion);
        
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-parent-validation');
        fs.symlinkSync(tempDir, originalSpecsPath);
        
        try {
          parseAllSpecs();
          assert.fail('Should have thrown error for invalid parent reference');
        } catch (error) {
          assert.ok(error.message.includes("Parent spec 'nonexistent-spec' not found"), 
            'Should detect invalid parent reference');
          assert.ok(error.message.includes('test-invalid-parent'), 
            'Error should mention the assertion ID');
        } finally {
          fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });
  });

  describe('Error Message Quality', () => {
    test('provides clear, actionable error messages', () => {
      // Test various error conditions and their messages
      const testCases = [
        {
          name: 'missing id field',
          specContent: `---
created: 2026-01-20T16:00:00Z
priority: 1
---

# Missing ID Test`,
          expectedError: "Missing required field 'id'",
          fileName: 'temp-missing-id'
        },
        {
          name: 'invalid timestamp',
          specContent: `---
id: temp-invalid-timestamp
created: 2026-01-20
priority: 1
---

# Invalid Timestamp Test`,
          expectedError: "Invalid ISO 8601 timestamp in 'created' field: '2026-01-20'",
          fileName: 'temp-invalid-timestamp'
        },
        {
          name: 'invalid priority',
          specContent: `---
id: temp-invalid-priority
created: 2026-01-20T16:00:00Z
priority: 0
---

# Invalid Priority Test`,
          expectedError: "Invalid priority value '0' (must be: 1, 2, or 3)",
          fileName: 'temp-invalid-priority'
        }
      ];
      
      for (const testCase of testCases) {
        const tempDir = path.join(process.cwd(), testCase.fileName);
        
        try {
          fs.mkdirSync(tempDir, { recursive: true });
          fs.mkdirSync(path.join(tempDir, 'assertions'), { recursive: true });
          
          // Write the spec file with proper naming convention
          fs.writeFileSync(path.join(tempDir, `${testCase.fileName}.md`), testCase.specContent);
          
          const originalSpecsPath = path.join(process.cwd(), 'specs', testCase.fileName);
          fs.symlinkSync(tempDir, originalSpecsPath);
          
          try {
            parseAllSpecs();
            assert.fail(`Should have thrown error for ${testCase.name}`);
          } catch (error) {
            assert.ok(error.message.includes(testCase.expectedError), 
              `Error for ${testCase.name} should include: ${testCase.expectedError}. Got: ${error.message}`);
            assert.ok(error.message.includes(`${testCase.fileName}.md`), 
              `Error for ${testCase.name} should include file name: ${testCase.fileName}.md`);
          } finally {
            if (fs.existsSync(originalSpecsPath)) {
              fs.unlinkSync(originalSpecsPath);
            }
          }
          
        } finally {
          if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
        }
      }
    });
  });
});

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

describe('Next Priority Identification', () => {
  describe('Priority Algorithm', () => {
    test('identifies highest priority incomplete assertion', () => {
      const tempDir1 = path.join(process.cwd(), 'temp-priority-test-1');
      const tempDir2 = path.join(process.cwd(), 'temp-priority-test-2');
      const assertionsDir1 = path.join(tempDir1, 'assertions');
      const assertionsDir2 = path.join(tempDir2, 'assertions');
      
      try {
        fs.mkdirSync(tempDir1, { recursive: true });
        fs.mkdirSync(assertionsDir1, { recursive: true });
        fs.mkdirSync(tempDir2, { recursive: true });
        fs.mkdirSync(assertionsDir2, { recursive: true });
        
        // Create first spec with priority 2 assertions
        const spec1Content = `---
id: temp-priority-test-1
created: 2026-01-20T16:00:00Z
priority: 2
---

# Priority Test Spec 1

Test spec for priority algorithm.`;
        
        fs.writeFileSync(path.join(tempDir1, 'temp-priority-test-1.md'), spec1Content);
        
        // Priority 2 assertion (not_started)
        const assertion1 = `---
id: low-priority-assertion
parent: temp-priority-test-1
created: 2026-01-20T16:00:00Z
priority: 2
status: not_started
---

# Low Priority Assertion

This should not be picked first.`;
        
        fs.writeFileSync(path.join(assertionsDir1, 'low-priority.md'), assertion1);
        
        // Create second spec with priority 1 assertion
        const spec2Content = `---
id: temp-priority-test-2
created: 2026-01-20T16:00:00Z
priority: 1
---

# Priority Test Spec 2

Test spec for priority algorithm.`;
        
        fs.writeFileSync(path.join(tempDir2, 'temp-priority-test-2.md'), spec2Content);
        
        // Priority 1 assertion (not_started) - should be picked first
        const assertion2 = `---
id: high-priority-assertion
parent: temp-priority-test-2
created: 2026-01-20T16:01:00Z
priority: 1
status: not_started
---

# High Priority Assertion

This should be picked first.`;
        
        fs.writeFileSync(path.join(assertionsDir2, 'high-priority.md'), assertion2);
        
        const originalSpecsPath1 = path.join(process.cwd(), 'specs', 'temp-priority-test-1');
        const originalSpecsPath2 = path.join(process.cwd(), 'specs', 'temp-priority-test-2');
        
        fs.symlinkSync(tempDir1, originalSpecsPath1);
        fs.symlinkSync(tempDir2, originalSpecsPath2);
        
        try {
          const { assertions } = parseAllSpecs();
          const nextAssertion = findNextAssertion(assertions);
          
          assert.ok(nextAssertion, 'Should find a next assertion');
          assert.equal(nextAssertion.id, 'high-priority-assertion', 'Should pick priority 1 assertion over priority 2');
          assert.equal(nextAssertion.priority, 1, 'Selected assertion should have priority 1');
          
        } finally {
          fs.unlinkSync(originalSpecsPath1);
          fs.unlinkSync(originalSpecsPath2);
        }
        
      } finally {
        if (fs.existsSync(tempDir1)) fs.rmSync(tempDir1, { recursive: true });
        if (fs.existsSync(tempDir2)) fs.rmSync(tempDir2, { recursive: true });
      }
    });

    test('breaks ties by oldest created timestamp', () => {
      const tempDir = path.join(process.cwd(), 'temp-tie-breaker-test');
      const assertionsDir = path.join(tempDir, 'assertions');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(assertionsDir, { recursive: true });
        
        const specContent = `---
id: temp-tie-breaker-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Tie Breaker Test Spec

Test spec for timestamp tie-breaking.`;
        
        fs.writeFileSync(path.join(tempDir, 'temp-tie-breaker-test.md'), specContent);
        
        // Create assertions with same priority but different timestamps
        const newerAssertion = `---
id: newer-assertion
parent: temp-tie-breaker-test
created: 2026-01-20T16:01:00Z
priority: 1
status: not_started
---

# Newer Assertion

This was created later.`;
        
        const olderAssertion = `---
id: older-assertion
parent: temp-tie-breaker-test
created: 2026-01-20T15:59:00Z
priority: 1
status: not_started
---

# Older Assertion

This was created earlier and should be picked.`;
        
        fs.writeFileSync(path.join(assertionsDir, 'newer.md'), newerAssertion);
        fs.writeFileSync(path.join(assertionsDir, 'older.md'), olderAssertion);
        
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-tie-breaker-test');
        fs.symlinkSync(tempDir, originalSpecsPath);
        
        try {
          const { assertions } = parseAllSpecs();
          const nextAssertion = findNextAssertion(assertions);
          
          assert.ok(nextAssertion, 'Should find a next assertion');
          assert.equal(nextAssertion.id, 'older-assertion', 'Should pick older assertion when priorities are equal');
          assert.equal(nextAssertion.created, '2026-01-20T15:59:00Z', 'Selected assertion should be the older one');
          
        } finally {
          fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });

    test('filters out done assertions', () => {
      const tempDir = path.join(process.cwd(), 'temp-done-filter-test');
      const assertionsDir = path.join(tempDir, 'assertions');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(assertionsDir, { recursive: true });
        
        const specContent = `---
id: temp-done-filter-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Done Filter Test Spec

Test spec for filtering done assertions.`;
        
        fs.writeFileSync(path.join(tempDir, 'temp-done-filter-test.md'), specContent);
        
        // Done assertion (should be filtered out)
        const doneAssertion = `---
id: done-assertion
parent: temp-done-filter-test
created: 2026-01-20T15:59:00Z
priority: 1
status: done
---

# Done Assertion

This is complete and should be ignored.`;
        
        // Not started assertion (should be picked)
        const notStartedAssertion = `---
id: not-started-assertion
parent: temp-done-filter-test
created: 2026-01-20T16:01:00Z
priority: 2
status: not_started
---

# Not Started Assertion

This should be picked even though it has lower priority.`;
        
        fs.writeFileSync(path.join(assertionsDir, 'done.md'), doneAssertion);
        fs.writeFileSync(path.join(assertionsDir, 'not-started.md'), notStartedAssertion);
        
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-done-filter-test');
        fs.symlinkSync(tempDir, originalSpecsPath);
        
        try {
          const { assertions } = parseAllSpecs();
          const nextAssertion = findNextAssertion(assertions);
          
          assert.ok(nextAssertion, 'Should find a next assertion');
          assert.equal(nextAssertion.id, 'not-started-assertion', 'Should pick incomplete assertion over done ones');
          assert.equal(nextAssertion.status, 'not_started', 'Selected assertion should be not_started');
          
        } finally {
          fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });

    test('includes in_progress assertions in incomplete filter', () => {
      const tempDir = path.join(process.cwd(), 'temp-in-progress-test');
      const assertionsDir = path.join(tempDir, 'assertions');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(assertionsDir, { recursive: true });
        
        const specContent = `---
id: temp-in-progress-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# In Progress Test Spec

Test spec for in_progress status handling.`;
        
        fs.writeFileSync(path.join(tempDir, 'temp-in-progress-test.md'), specContent);
        
        // In progress assertion (should be picked)
        const inProgressAssertion = `---
id: in-progress-assertion
parent: temp-in-progress-test
created: 2026-01-20T15:59:00Z
priority: 1
status: in_progress
---

# In Progress Assertion

This is in progress and should be picked up.`;
        
        fs.writeFileSync(path.join(assertionsDir, 'in-progress.md'), inProgressAssertion);
        
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-in-progress-test');
        fs.symlinkSync(tempDir, originalSpecsPath);
        
        try {
          const { assertions } = parseAllSpecs();
          const nextAssertion = findNextAssertion(assertions);
          
          assert.ok(nextAssertion, 'Should find a next assertion');
          assert.equal(nextAssertion.id, 'in-progress-assertion', 'Should pick in_progress assertion');
          assert.equal(nextAssertion.status, 'in_progress', 'Selected assertion should be in_progress');
          
        } finally {
          fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });

    test('returns null when all test assertions are done', () => {
      // Test the findNextAssertion function directly with controlled data
      const testAssertions = [
        {
          id: 'done-assertion-1',
          parent: 'test-spec',
          priority: 1,
          status: 'done',
          created: '2026-01-20T15:59:00Z'
        },
        {
          id: 'done-assertion-2',
          parent: 'test-spec',
          priority: 2,
          status: 'done',
          created: '2026-01-20T16:01:00Z'
        }
      ];
      
      const nextAssertion = findNextAssertion(testAssertions);
      assert.equal(nextAssertion, null, 'Should return null when all assertions are done');
    });
  });

  describe('CLI Integration', () => {
    test('npm run next returns valid JSON with next assertion', () => {
      const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
      const parsed = JSON.parse(result);
      
      // Should return either an assertion or completion status
      if (parsed.type === 'assertion') {
        assert.ok(parsed.id, 'Should have assertion id');
        assert.ok(parsed.parent, 'Should have parent spec');
        assert.ok(parsed.file, 'Should have file path');
        assert.ok([1, 2, 3].includes(parsed.priority), 'Should have valid priority');
        assert.ok(['not_started', 'in_progress'].includes(parsed.status), 'Should be incomplete');
        assert.ok(parsed.title, 'Should have title');
        assert.ok(parsed.content, 'Should have content');
        assert.ok(parsed.spec, 'Should have spec reference');
      } else if (parsed.status === 'complete') {
        assert.ok(parsed.message, 'Complete status should have message');
      } else if (parsed.status === 'empty') {
        assert.ok(parsed.message, 'Empty status should have message');
      }
    });

    test('CLI output matches priority algorithm', () => {
      // Create controlled test environment
      const tempDir1 = path.join(process.cwd(), 'temp-cli-test-1');
      const tempDir2 = path.join(process.cwd(), 'temp-cli-test-2');
      const assertionsDir1 = path.join(tempDir1, 'assertions');
      const assertionsDir2 = path.join(tempDir2, 'assertions');
      
      try {
        fs.mkdirSync(tempDir1, { recursive: true });
        fs.mkdirSync(assertionsDir1, { recursive: true });
        fs.mkdirSync(tempDir2, { recursive: true });
        fs.mkdirSync(assertionsDir2, { recursive: true });
        
        // Create specs and assertions in controlled order
        const spec1Content = `---
id: temp-cli-test-1
created: 2026-01-20T16:00:00Z
priority: 1
---

# CLI Test Spec 1`;
        
        const spec2Content = `---
id: temp-cli-test-2
created: 2026-01-20T16:00:00Z
priority: 2
---

# CLI Test Spec 2`;
        
        fs.writeFileSync(path.join(tempDir1, 'temp-cli-test-1.md'), spec1Content);
        fs.writeFileSync(path.join(tempDir2, 'temp-cli-test-2.md'), spec2Content);
        
        // Priority 2 assertion (should not be picked)
        const lowPriorityAssertion = `---
id: cli-low-priority
parent: temp-cli-test-2
created: 2026-01-20T15:59:00Z
priority: 2
status: not_started
---

# CLI Low Priority Assertion`;
        
        // Priority 1 assertion (should be picked)
        const highPriorityAssertion = `---
id: cli-high-priority
parent: temp-cli-test-1
created: 2026-01-20T16:01:00Z
priority: 1
status: not_started
---

# CLI High Priority Assertion`;
        
        fs.writeFileSync(path.join(assertionsDir2, 'low-priority.md'), lowPriorityAssertion);
        fs.writeFileSync(path.join(assertionsDir1, 'high-priority.md'), highPriorityAssertion);
        
        const originalSpecsPath1 = path.join(process.cwd(), 'specs', 'temp-cli-test-1');
        const originalSpecsPath2 = path.join(process.cwd(), 'specs', 'temp-cli-test-2');
        
        fs.symlinkSync(tempDir1, originalSpecsPath1);
        fs.symlinkSync(tempDir2, originalSpecsPath2);
        
        try {
          // Test direct function call
          const { assertions } = parseAllSpecs();
          const nextAssertion = findNextAssertion(assertions);
          
          // Test CLI output
          const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
          const parsed = JSON.parse(result);
          
          // CLI should match direct function call
          if (parsed.type === 'assertion') {
            assert.equal(parsed.id, nextAssertion.id, 'CLI should return same assertion as direct function call');
            assert.equal(parsed.priority, nextAssertion.priority, 'CLI priority should match function result');
          }
          
        } finally {
          fs.unlinkSync(originalSpecsPath1);
          fs.unlinkSync(originalSpecsPath2);
        }
        
      } finally {
        if (fs.existsSync(tempDir1)) fs.rmSync(tempDir1, { recursive: true });
        if (fs.existsSync(tempDir2)) fs.rmSync(tempDir2, { recursive: true });
      }
    });
  });
});