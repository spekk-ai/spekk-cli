import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'fs';
import path from 'path';
import { parseAllSpecs, parseFrontmatter, validateFields, findNextAssertion } from '../index.js';

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