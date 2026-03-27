import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'fs';
import path from 'path';
import os from 'os';
import * as parserModule from '../index.js';

const { parseAllSpecs, findNextAssertion } = parserModule;

describe('Parser Basic Tests', () => {
  test('identifies highest priority assertion as next', () => {
    const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-basic-'));
    const specsDir = path.join(testDir, 'specs');
    const specDir = path.join(specsDir, 'temp-priority-basic-test');
    const assertionsDir = path.join(specDir, 'assertions');

    try {
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(specDir, 'temp-priority-basic-test.md'), `---
id: temp-priority-basic-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Priority Test Spec`);

      fs.writeFileSync(path.join(assertionsDir, 'high-priority.md'), `---
id: high-priority-assertion
parent: temp-priority-basic-test
created: 2026-01-20T15:59:00Z
priority: 1
status: not_started
---

# High Priority Assertion`);

      fs.writeFileSync(path.join(assertionsDir, 'low-priority.md'), `---
id: low-priority-assertion
parent: temp-priority-basic-test
created: 2026-01-20T15:58:00Z
priority: 2
status: not_started
---

# Low Priority Assertion`);

      const { assertions } = parseAllSpecs(specsDir);
      const nextAssertion = findNextAssertion(assertions, [], { allBranches: true });

      assert.ok(nextAssertion, 'Should find next assertion');
      assert.equal(nextAssertion.id, 'high-priority-assertion', 'Should pick priority 1 over priority 2');
    } finally {
      if (fs.existsSync(testDir)) fs.rmSync(testDir, { recursive: true, force: true });
    }
  });

  test('filters out done assertions from next selection', () => {
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

    const nextAssertion = findNextAssertion(testAssertions, [], { allBranches: true });
    assert.ok(nextAssertion, 'Should find next assertion');
    assert.equal(nextAssertion.id, 'not-started-assertion', 'Should skip done assertions');
  });

  test('enforces proper folder structure for specs', () => {
    const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-structure-'));
    const specsDir = path.join(testDir, 'specs');
    const specDir = path.join(specsDir, 'temp-valid-structure-test');
    const assertionsDir = path.join(specDir, 'assertions');

    try {
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(specDir, 'temp-valid-structure-test.md'), `---
id: temp-valid-structure-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Valid Structure Test Spec`);

      fs.writeFileSync(path.join(assertionsDir, 'assertion.md'), `---
id: valid-structure-assertion
parent: temp-valid-structure-test
created: 2026-01-20T16:00:00Z
priority: 1
status: not_started
---

# Valid Structure Assertion`);

      const { specs, assertions } = parseAllSpecs(specsDir);
      const testSpec = specs.find(s => s.id === 'temp-valid-structure-test');
      const testAssertion = assertions.find(a => a.id === 'valid-structure-assertion');

      assert.ok(testSpec, 'Should find properly structured spec');
      assert.ok(testAssertion, 'Should find assertion in proper directory');
      assert.equal(testAssertion.parent, 'temp-valid-structure-test', 'Assertion should reference parent spec');
    } finally {
      if (fs.existsSync(testDir)) fs.rmSync(testDir, { recursive: true, force: true });
    }
  });
});
