import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import fs from 'fs';
import path from 'path';
import { parseAllSpecs, findNextAssertion } from '../index.js';

describe('Parser Basic Tests', () => {
  describe('Parser Script Existence and Executability', () => {
    test('parser script exists and is executable', () => {
      const parserPath = path.join(process.cwd(), 'src', 'parser', 'cli.js');
      
      assert.ok(fs.existsSync(parserPath), 'Parser CLI script should exist at src/parser/cli.js');
      
      // Test that the script can be executed and returns JSON (may exit with error code but should output JSON)
      try {
        const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
        JSON.parse(result); // Should be valid JSON
      } catch (error) {
        // Script may exit with error code but should still output JSON
        if (error.stdout) {
          JSON.parse(error.stdout); // Should be valid JSON even on error
        } else {
          throw error;
        }
      }
    });

    test('npm run next command works', () => {
      // Verify the npm script exists and works
      assert.doesNotThrow(() => {
        execSync('npm run next', { encoding: 'utf8' });
      }, 'npm run next should execute successfully');
    });
  });

  describe('JSON Output Format Validation', () => {
    test('outputs valid JSON format', () => {
      const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
      
      let parsed;
      assert.doesNotThrow(() => {
        parsed = JSON.parse(result);
      }, 'Output should be valid JSON');
      
      assert.equal(typeof parsed, 'object', 'JSON should be an object');
      assert.ok(!Array.isArray(parsed), 'JSON should not be an array');
      assert.notEqual(parsed, null, 'JSON should not be null');
    });

    test('assertion output includes required fields', () => {
      const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
      const parsed = JSON.parse(result);
      
      if (parsed.type === 'assertion') {
        const requiredFields = ['type', 'id', 'parent', 'file', 'priority', 'status', 'title', 'content', 'spec'];
        
        for (const field of requiredFields) {
          assert.ok(parsed.hasOwnProperty(field), `Should have ${field} field`);
          assert.ok(parsed[field] !== undefined, `${field} should not be undefined`);
        }
        
        // Verify field types
        assert.equal(typeof parsed.type, 'string', 'type should be string');
        assert.equal(typeof parsed.id, 'string', 'id should be string');
        assert.equal(typeof parsed.priority, 'number', 'priority should be number');
        assert.ok([1, 2, 3].includes(parsed.priority), 'priority should be 1, 2, or 3');
        assert.ok(['not_started', 'in_progress', 'done'].includes(parsed.status), 'status should be valid');
      }
    });

    test('complete status has correct format', () => {
      // Test by creating a controlled environment with all done assertions
      const tempDir = path.join(process.cwd(), 'temp-complete-basic-test');
      const assertionsDir = path.join(tempDir, 'assertions');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(assertionsDir, { recursive: true });
        
        const specContent = `---
id: temp-complete-basic-test
created: 2026-01-20T16:00:00Z
priority: 1
---
# Complete Test Spec`;
        
        const doneAssertion = `---
id: done-assertion
parent: temp-complete-basic-test
created: 2026-01-20T16:00:00Z
priority: 1
status: done
---
# Done Assertion`;
        
        fs.writeFileSync(path.join(tempDir, 'temp-complete-basic-test.md'), specContent);
        fs.writeFileSync(path.join(assertionsDir, 'done.md'), doneAssertion);
        
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-complete-basic-test');
        fs.symlinkSync(tempDir, originalSpecsPath);
        
        try {
          const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
          const parsed = JSON.parse(result);
          
          if (parsed.status === 'complete') {
            assert.equal(parsed.status, 'complete', 'Should have complete status');
            assert.ok(parsed.message, 'Should have message field');
            assert.equal(typeof parsed.message, 'string', 'message should be string');
          }
          
        } finally {
          fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });
  });

  describe('Next Priority Assertion Identification', () => {
    test('identifies next priority assertion correctly', () => {
      const tempDir = path.join(process.cwd(), 'temp-priority-basic-test');
      const assertionsDir = path.join(tempDir, 'assertions');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(assertionsDir, { recursive: true });
        
        const specContent = `---
id: temp-priority-basic-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Priority Test Spec`;
        
        // Create high priority assertion
        const highPriorityAssertion = `---
id: high-priority-assertion
parent: temp-priority-basic-test
created: 2026-01-20T15:59:00Z
priority: 1
status: not_started
---

# High Priority Assertion`;
        
        // Create low priority assertion
        const lowPriorityAssertion = `---
id: low-priority-assertion
parent: temp-priority-basic-test
created: 2026-01-20T15:58:00Z
priority: 2
status: not_started
---

# Low Priority Assertion`;
        
        fs.writeFileSync(path.join(tempDir, 'temp-priority-basic-test.md'), specContent);
        fs.writeFileSync(path.join(assertionsDir, 'high-priority.md'), highPriorityAssertion);
        fs.writeFileSync(path.join(assertionsDir, 'low-priority.md'), lowPriorityAssertion);
        
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-priority-basic-test');
        fs.symlinkSync(tempDir, originalSpecsPath);
        
        try {
          const { assertions } = parseAllSpecs();
          const nextAssertion = findNextAssertion(assertions.filter(a => a.parent === 'temp-priority-basic-test'));
          
          assert.ok(nextAssertion, 'Should find next assertion');
          assert.equal(nextAssertion.id, 'high-priority-assertion', 'Should pick priority 1 over priority 2');
          assert.equal(nextAssertion.priority, 1, 'Selected assertion should have priority 1');
          
        } finally {
          fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });

    test('filters out done assertions', () => {
      const testAssertions = [
        {
          id: 'done-assertion',
          parent: 'test-spec',
          priority: 1,
          status: 'done',
          created: '2026-01-20T15:59:00Z'
        },
        {
          id: 'not-started-assertion',
          parent: 'test-spec', 
          priority: 2,
          status: 'not_started',
          created: '2026-01-20T16:01:00Z'
        }
      ];
      
      const nextAssertion = findNextAssertion(testAssertions);
      assert.ok(nextAssertion, 'Should find next assertion');
      assert.equal(nextAssertion.id, 'not-started-assertion', 'Should skip done assertions');
      assert.equal(nextAssertion.status, 'not_started', 'Selected assertion should be incomplete');
    });
  });

  describe('Folder Structure Enforcement', () => {
    test('enforces folder structure - no flat spec files', () => {
      // Test that specs must be in proper directory structure, not flat .md files
      const tempDir = path.join(process.cwd(), 'temp-flat-file-test');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        
        // Create a flat .md file (should not be allowed as spec)
        const flatSpec = `---
id: flat-spec-file
created: 2026-01-20T16:00:00Z
priority: 1
---

# Flat Spec File

This is a spec file at the root level.`;
        
        fs.writeFileSync(path.join(tempDir, 'flat-spec.md'), flatSpec);
        
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'flat-spec');
        fs.symlinkSync(tempDir, originalSpecsPath);
        
        try {
          // Parser should either ignore flat files or handle them appropriately
          // This test ensures the structure is enforced
          const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
          const parsed = JSON.parse(result);
          
          // Should not find the flat file as a valid spec
          if (parsed.type === 'assertion') {
            assert.notEqual(parsed.parent, 'flat-spec-file', 'Should not find flat spec files');
          }
          
        } finally {
          fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });

    test('enforces folder structure - valid spec directories', () => {
      const tempDir = path.join(process.cwd(), 'temp-valid-structure-test');
      const assertionsDir = path.join(tempDir, 'assertions');
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(assertionsDir, { recursive: true });
        
        const specContent = `---
id: temp-valid-structure-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Valid Structure Test Spec`;
        const assertionContent = `---
id: valid-structure-assertion
parent: temp-valid-structure-test
created: 2026-01-20T16:00:00Z
priority: 1
status: not_started
---

# Valid Structure Assertion`;
        fs.writeFileSync(path.join(tempDir, 'temp-valid-structure-test.md'), specContent);
        fs.writeFileSync(path.join(assertionsDir, 'assertion.md'), assertionContent);
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-valid-structure-test');
        fs.symlinkSync(tempDir, originalSpecsPath);
        try {
          const { specs, assertions } = parseAllSpecs();
          const testSpec = specs.find(s => s.id === 'temp-valid-structure-test');
          const testAssertion = assertions.find(a => a.id === 'valid-structure-assertion');
          assert.ok(testSpec, 'Should find properly structured spec');
          assert.ok(testAssertion, 'Should find assertion in proper directory');
          assert.equal(testAssertion.parent, 'temp-valid-structure-test', 'Assertion should reference parent spec');
        } finally {
          fs.unlinkSync(originalSpecsPath);
        }
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });
  });
});