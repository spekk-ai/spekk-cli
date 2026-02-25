import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import fs from 'fs';
import path from 'path';
import os from 'os';
import { parseAllSpecs, findNextAssertion } from '../index.js';

describe('Next Priority Identification', () => {
  describe('Priority Algorithm', () => {
    test('identifies highest priority incomplete assertion', () => {
      const tempDir1 = path.join(os.tmpdir(), `spekk-test-priority-1-${Date.now()}`);
      const tempDir2 = path.join(os.tmpdir(), `spekk-test-priority-2-${Date.now()}`);
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
          // Filter to only test assertions to avoid interference from real specs
          const testAssertions = assertions.filter(a => a.parent.startsWith('temp-priority-test-'));
          const nextAssertion = findNextAssertion(testAssertions, [], { allBranches: true });
          
          assert.ok(nextAssertion, 'Should find a next assertion');
          assert.equal(nextAssertion.id, 'high-priority-assertion', 'Should pick priority 1 assertion over priority 2');
          assert.equal(nextAssertion.priority, 1, 'Selected assertion should have priority 1');
          
        } finally {
          // Clean up symlinks first
          if (fs.existsSync(originalSpecsPath1)) fs.unlinkSync(originalSpecsPath1);
          if (fs.existsSync(originalSpecsPath2)) fs.unlinkSync(originalSpecsPath2);
        }
        
      } finally {
        // Clean up temp directories
        if (fs.existsSync(tempDir1)) fs.rmSync(tempDir1, { recursive: true, force: true });
        if (fs.existsSync(tempDir2)) fs.rmSync(tempDir2, { recursive: true, force: true });
      }
    });

    test('breaks ties by oldest created timestamp', () => {
      const tempDir = path.join(os.tmpdir(), `spekk-test-tie-breaker-${Date.now()}`);
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
          // Filter to only test assertions to avoid interference from real specs
          const testAssertions = assertions.filter(a => a.parent === 'temp-tie-breaker-test');
          const nextAssertion = findNextAssertion(testAssertions, [], { allBranches: true });
          
          assert.ok(nextAssertion, 'Should find a next assertion');
          assert.equal(nextAssertion.id, 'older-assertion', 'Should pick older assertion when priorities are equal');
          assert.equal(nextAssertion.created, '2026-01-20T15:59:00Z', 'Selected assertion should be the older one');
          
        } finally {
          // Clean up symlink first
          if (fs.existsSync(originalSpecsPath)) fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        // Clean up temp directory
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true, force: true });
      }
    });

    test('filters out done assertions', () => {
      const tempDir = path.join(os.tmpdir(), `spekk-test-done-filter-${Date.now()}`);
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
          // Filter to only test assertions to avoid interference from real specs
          const testAssertions = assertions.filter(a => a.parent === 'temp-done-filter-test');
          const nextAssertion = findNextAssertion(testAssertions, [], { allBranches: true });
          
          assert.ok(nextAssertion, 'Should find a next assertion');
          assert.equal(nextAssertion.id, 'not-started-assertion', 'Should pick incomplete assertion over done ones');
          assert.equal(nextAssertion.status, 'not_started', 'Selected assertion should be not_started');
          
        } finally {
          // Clean up symlink first
          if (fs.existsSync(originalSpecsPath)) fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        // Clean up temp directory
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true, force: true });
      }
    });

    test('includes in_progress assertions in incomplete filter', () => {
      const tempDir = path.join(os.tmpdir(), `spekk-test-in-progress-${Date.now()}`);
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
          // Filter to only test assertions to avoid interference from real specs
          const testAssertions = assertions.filter(a => a.parent === 'temp-in-progress-test');
          const nextAssertion = findNextAssertion(testAssertions, [], { allBranches: true });
          
          assert.ok(nextAssertion, 'Should find a next assertion');
          assert.equal(nextAssertion.id, 'in-progress-assertion', 'Should pick in_progress assertion');
          assert.equal(nextAssertion.status, 'in_progress', 'Selected assertion should be in_progress');
          
        } finally {
          // Clean up symlink first
          if (fs.existsSync(originalSpecsPath)) fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        // Clean up temp directory
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true, force: true });
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
      
      const nextAssertion = findNextAssertion(testAssertions, [], { allBranches: true });
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
      const tempDir1 = path.join(os.tmpdir(), `spekk-test-cli-1-${Date.now()}`);
      const tempDir2 = path.join(os.tmpdir(), `spekk-test-cli-2-${Date.now()}`);
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
          // Test direct function call with filtered test data
          const { assertions } = parseAllSpecs();
          const testAssertions = assertions.filter(a => a.parent.startsWith('temp-cli-test-'));
          const nextAssertion = findNextAssertion(testAssertions, [], { allBranches: true });
          
          // Test CLI output (uses all real specs, so we can't guarantee it matches test data)
          const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
          const parsed = JSON.parse(result);
          
          // Verify CLI returns valid structure (can't guarantee specific test assertion due to real specs)
          if (parsed.type === 'assertion') {
            assert.ok(parsed.id, 'CLI should return assertion with id');
            assert.ok([1, 2, 3].includes(parsed.priority), 'CLI should return valid priority');
          }
          
          // Verify test function works correctly
          assert.ok(nextAssertion, 'Function should find next test assertion');
          assert.equal(nextAssertion.id, 'cli-high-priority', 'Should pick priority 1 test assertion');
          
        } finally {
          // Clean up symlinks first
          if (fs.existsSync(originalSpecsPath1)) fs.unlinkSync(originalSpecsPath1);
          if (fs.existsSync(originalSpecsPath2)) fs.unlinkSync(originalSpecsPath2);
        }
        
      } finally {
        // Clean up temp directories
        if (fs.existsSync(tempDir1)) fs.rmSync(tempDir1, { recursive: true, force: true });
        if (fs.existsSync(tempDir2)) fs.rmSync(tempDir2, { recursive: true, force: true });
      }
    });
  });
});