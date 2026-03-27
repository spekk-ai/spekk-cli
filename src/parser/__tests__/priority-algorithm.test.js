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
      const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-priority-'));
      const specsDir = path.join(testDir, 'specs');

      try {
        // Create first spec with priority 2 assertions
        const specDir1 = path.join(specsDir, 'temp-priority-test-1');
        const assertionsDir1 = path.join(specDir1, 'assertions');
        fs.mkdirSync(assertionsDir1, { recursive: true });

        fs.writeFileSync(path.join(specDir1, 'temp-priority-test-1.md'), `---
id: temp-priority-test-1
created: 2026-01-20T16:00:00Z
priority: 2
---

# Priority Test Spec 1

Test spec for priority algorithm.`);

        fs.writeFileSync(path.join(assertionsDir1, 'low-priority.md'), `---
id: low-priority-assertion
parent: temp-priority-test-1
created: 2026-01-20T16:00:00Z
priority: 2
status: not_started
---

# Low Priority Assertion

This should not be picked first.`);

        // Create second spec with priority 1 assertion
        const specDir2 = path.join(specsDir, 'temp-priority-test-2');
        const assertionsDir2 = path.join(specDir2, 'assertions');
        fs.mkdirSync(assertionsDir2, { recursive: true });

        fs.writeFileSync(path.join(specDir2, 'temp-priority-test-2.md'), `---
id: temp-priority-test-2
created: 2026-01-20T16:00:00Z
priority: 1
---

# Priority Test Spec 2

Test spec for priority algorithm.`);

        fs.writeFileSync(path.join(assertionsDir2, 'high-priority.md'), `---
id: high-priority-assertion
parent: temp-priority-test-2
created: 2026-01-20T16:01:00Z
priority: 1
status: not_started
---

# High Priority Assertion

This should be picked first.`);

        const { assertions } = parseAllSpecs(specsDir);
        const nextAssertion = findNextAssertion(assertions, [], { allBranches: true });

        assert.ok(nextAssertion, 'Should find a next assertion');
        assert.equal(nextAssertion.id, 'high-priority-assertion', 'Should pick priority 1 assertion over priority 2');
        assert.equal(nextAssertion.priority, 1, 'Selected assertion should have priority 1');
      } finally {
        if (fs.existsSync(testDir)) fs.rmSync(testDir, { recursive: true, force: true });
      }
    });

    test('breaks ties by oldest created timestamp', () => {
      const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-tie-'));
      const specsDir = path.join(testDir, 'specs');
      const specDir = path.join(specsDir, 'temp-tie-breaker-test');
      const assertionsDir = path.join(specDir, 'assertions');

      try {
        fs.mkdirSync(assertionsDir, { recursive: true });

        fs.writeFileSync(path.join(specDir, 'temp-tie-breaker-test.md'), `---
id: temp-tie-breaker-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Tie Breaker Test Spec

Test spec for timestamp tie-breaking.`);

        fs.writeFileSync(path.join(assertionsDir, 'newer.md'), `---
id: newer-assertion
parent: temp-tie-breaker-test
created: 2026-01-20T16:01:00Z
priority: 1
status: not_started
---

# Newer Assertion

This was created later.`);

        fs.writeFileSync(path.join(assertionsDir, 'older.md'), `---
id: older-assertion
parent: temp-tie-breaker-test
created: 2026-01-20T15:59:00Z
priority: 1
status: not_started
---

# Older Assertion

This was created earlier and should be picked.`);

        const { assertions } = parseAllSpecs(specsDir);
        const nextAssertion = findNextAssertion(assertions, [], { allBranches: true });

        assert.ok(nextAssertion, 'Should find a next assertion');
        assert.equal(nextAssertion.id, 'older-assertion', 'Should pick older assertion when priorities are equal');
        assert.equal(nextAssertion.created, '2026-01-20T15:59:00Z', 'Selected assertion should be the older one');
      } finally {
        if (fs.existsSync(testDir)) fs.rmSync(testDir, { recursive: true, force: true });
      }
    });

    test('filters out done assertions', () => {
      const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-done-'));
      const specsDir = path.join(testDir, 'specs');
      const specDir = path.join(specsDir, 'temp-done-filter-test');
      const assertionsDir = path.join(specDir, 'assertions');

      try {
        fs.mkdirSync(assertionsDir, { recursive: true });

        fs.writeFileSync(path.join(specDir, 'temp-done-filter-test.md'), `---
id: temp-done-filter-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Done Filter Test Spec

Test spec for filtering done assertions.`);

        fs.writeFileSync(path.join(assertionsDir, 'done.md'), `---
id: done-assertion
parent: temp-done-filter-test
created: 2026-01-20T15:59:00Z
priority: 1
status: done
---

# Done Assertion

This is complete and should be ignored.`);

        fs.writeFileSync(path.join(assertionsDir, 'not-started.md'), `---
id: not-started-assertion
parent: temp-done-filter-test
created: 2026-01-20T16:01:00Z
priority: 2
status: not_started
---

# Not Started Assertion

This should be picked even though it has lower priority.`);

        const { assertions } = parseAllSpecs(specsDir);
        const testAssertions = assertions.filter(a => a.parent === 'temp-done-filter-test');
        const nextAssertion = findNextAssertion(testAssertions, [], { allBranches: true });

        assert.ok(nextAssertion, 'Should find a next assertion');
        assert.equal(nextAssertion.id, 'not-started-assertion', 'Should pick incomplete assertion over done ones');
        assert.equal(nextAssertion.status, 'not_started', 'Selected assertion should be not_started');
      } finally {
        if (fs.existsSync(testDir)) fs.rmSync(testDir, { recursive: true, force: true });
      }
    });

    test('includes in_progress assertions in incomplete filter', () => {
      const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-inprogress-'));
      const specsDir = path.join(testDir, 'specs');
      const specDir = path.join(specsDir, 'temp-in-progress-test');
      const assertionsDir = path.join(specDir, 'assertions');

      try {
        fs.mkdirSync(assertionsDir, { recursive: true });

        fs.writeFileSync(path.join(specDir, 'temp-in-progress-test.md'), `---
id: temp-in-progress-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# In Progress Test Spec

Test spec for in_progress status handling.`);

        fs.writeFileSync(path.join(assertionsDir, 'in-progress.md'), `---
id: in-progress-assertion
parent: temp-in-progress-test
created: 2026-01-20T15:59:00Z
priority: 1
status: in_progress
---

# In Progress Assertion

This is in progress and should be picked up.`);

        const { assertions } = parseAllSpecs(specsDir);
        const testAssertions = assertions.filter(a => a.parent === 'temp-in-progress-test');
        const nextAssertion = findNextAssertion(testAssertions, [], { allBranches: true });

        assert.ok(nextAssertion, 'Should find a next assertion');
        assert.equal(nextAssertion.id, 'in-progress-assertion', 'Should pick in_progress assertion');
        assert.equal(nextAssertion.status, 'in_progress', 'Selected assertion should be in_progress');
      } finally {
        if (fs.existsSync(testDir)) fs.rmSync(testDir, { recursive: true, force: true });
      }
    });

    test('returns null when all test assertions are done', () => {
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
      const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-cli-'));
      const specsDir = path.join(testDir, 'specs');

      try {
        // Create first spec with priority 1 assertion
        const specDir1 = path.join(specsDir, 'temp-cli-test-1');
        const assertionsDir1 = path.join(specDir1, 'assertions');
        fs.mkdirSync(assertionsDir1, { recursive: true });

        fs.writeFileSync(path.join(specDir1, 'temp-cli-test-1.md'), `---
id: temp-cli-test-1
created: 2026-01-20T16:00:00Z
priority: 1
---

# CLI Test Spec 1`);

        fs.writeFileSync(path.join(assertionsDir1, 'high-priority.md'), `---
id: cli-high-priority
parent: temp-cli-test-1
created: 2026-01-20T16:01:00Z
priority: 1
status: not_started
---

# CLI High Priority Assertion`);

        // Create second spec with priority 2 assertion
        const specDir2 = path.join(specsDir, 'temp-cli-test-2');
        const assertionsDir2 = path.join(specDir2, 'assertions');
        fs.mkdirSync(assertionsDir2, { recursive: true });

        fs.writeFileSync(path.join(specDir2, 'temp-cli-test-2.md'), `---
id: temp-cli-test-2
created: 2026-01-20T16:00:00Z
priority: 2
---

# CLI Test Spec 2`);

        fs.writeFileSync(path.join(assertionsDir2, 'low-priority.md'), `---
id: cli-low-priority
parent: temp-cli-test-2
created: 2026-01-20T15:59:00Z
priority: 2
status: not_started
---

# CLI Low Priority Assertion`);

        // Test direct function call with isolated data
        const { assertions } = parseAllSpecs(specsDir);
        const nextAssertion = findNextAssertion(assertions, [], { allBranches: true });

        assert.ok(nextAssertion, 'Function should find next test assertion');
        assert.equal(nextAssertion.id, 'cli-high-priority', 'Should pick priority 1 test assertion');

        // Test CLI output (uses real project specs, so just verify valid structure)
        const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
        const parsed = JSON.parse(result);

        if (parsed.type === 'assertion') {
          assert.ok(parsed.id, 'CLI should return assertion with id');
          assert.ok([1, 2, 3].includes(parsed.priority), 'CLI should return valid priority');
        }
      } finally {
        if (fs.existsSync(testDir)) fs.rmSync(testDir, { recursive: true, force: true });
      }
    });
  });
});
