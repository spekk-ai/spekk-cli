import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'fs';
import path from 'path';
import * as parserModule from '../index.js';

const { parseAllSpecs, findNextAssertion } = parserModule;

describe('Parser Basic Tests', () => {
  test('identifies highest priority assertion as next', () => {
    const tempDir = path.join(process.cwd(), 'temp-priority-basic-test');
    const assertionsDir = path.join(tempDir, 'assertions');

    try {
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(tempDir, 'temp-priority-basic-test.md'), `---
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

      const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-priority-basic-test');
      fs.symlinkSync(tempDir, originalSpecsPath);

      try {
        const { assertions } = parseAllSpecs();
        const nextAssertion = findNextAssertion(assertions.filter(a => a.parent === 'temp-priority-basic-test'), [], { allBranches: true });

        assert.ok(nextAssertion, 'Should find next assertion');
        assert.equal(nextAssertion.id, 'high-priority-assertion', 'Should pick priority 1 over priority 2');

      } finally {
        fs.unlinkSync(originalSpecsPath);
      }

    } finally {
      if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
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
    const tempDir = path.join(process.cwd(), 'temp-valid-structure-test');
    const assertionsDir = path.join(tempDir, 'assertions');

    try {
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(tempDir, 'temp-valid-structure-test.md'), `---
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
