import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'fs';
import path from 'path';
import { validateFolderStructure, parseAllSpecs } from '../index.js';

describe('Parser skips specs without assertions directory', () => {
  test('validateFolderStructure does not throw when assertions/ is missing', () => {
    const tempDir = path.join(process.cwd(), 'temp-no-assertions-dir-test');
    const specDir = path.join(tempDir, 'my-spec');

    try {
      fs.mkdirSync(specDir, { recursive: true });

      // Create a valid parent spec file but no assertions/ directory
      fs.writeFileSync(path.join(specDir, 'my-spec.md'), `---
id: my-spec
created: 2026-01-28T21:35:00Z
priority: 1
---
# My Spec

A spec without an assertions directory.`);

      // Should NOT throw - it should warn and continue
      assert.doesNotThrow(() => {
        validateFolderStructure(tempDir);
      }, 'validateFolderStructure should not throw when assertions/ is missing');
    } finally {
      if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
    }
  });

  test('parseAllSpecs returns valid results when some specs lack assertions/', () => {
    const tempDir = path.join(process.cwd(), 'temp-mixed-specs-test');
    const completeSpecDir = path.join(tempDir, 'complete-spec');
    const incompleteSpecDir = path.join(tempDir, 'incomplete-spec');
    const assertionsDir = path.join(completeSpecDir, 'assertions');

    try {
      fs.mkdirSync(assertionsDir, { recursive: true });
      fs.mkdirSync(incompleteSpecDir, { recursive: true });

      // Complete spec with assertions/
      fs.writeFileSync(path.join(completeSpecDir, 'complete-spec.md'), `---
id: complete-spec
created: 2026-01-28T21:35:00Z
priority: 1
---
# Complete Spec

Has assertions directory.`);

      fs.writeFileSync(path.join(assertionsDir, 'an-assertion.md'), `---
id: an-assertion
parent: complete-spec
created: 2026-01-28T21:35:00Z
priority: 1
status: not_started
---
# An Assertion

This assertion should parse fine.`);

      // Incomplete spec - has parent .md but no assertions/
      fs.writeFileSync(path.join(incompleteSpecDir, 'incomplete-spec.md'), `---
id: incomplete-spec
created: 2026-01-28T21:35:00Z
priority: 2
---
# Incomplete Spec

Missing assertions directory.`);

      const { specs, assertions } = parseAllSpecs(tempDir);

      // The complete spec should be parsed
      const completeSpec = specs.find(s => s.id === 'complete-spec');
      assert.ok(completeSpec, 'Should parse the complete spec');

      // The assertion from the complete spec should be parsed
      const theAssertion = assertions.find(a => a.id === 'an-assertion');
      assert.ok(theAssertion, 'Should parse assertion from the complete spec');

      // The incomplete spec should still be parsed (it has a valid parent .md)
      // but treated as having zero assertions
      const incompleteSpec = specs.find(s => s.id === 'incomplete-spec');
      assert.ok(incompleteSpec, 'Incomplete spec should still be parsed');

      const incompleteAssertions = assertions.filter(a => a.parent === 'incomplete-spec');
      assert.equal(incompleteAssertions.length, 0, 'Incomplete spec should have zero assertions');
    } finally {
      if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
    }
  });

  test('spec directory with parent .md but no assertions/ is treated as zero assertions', () => {
    const tempDir = path.join(process.cwd(), 'temp-zero-assertions-test');
    const specDir = path.join(tempDir, 'lonely-spec');

    try {
      fs.mkdirSync(specDir, { recursive: true });

      fs.writeFileSync(path.join(specDir, 'lonely-spec.md'), `---
id: lonely-spec
created: 2026-01-28T21:35:00Z
priority: 1
---
# Lonely Spec

No assertions directory at all.`);

      // Should not throw
      const { specs, assertions } = parseAllSpecs(tempDir);

      const spec = specs.find(s => s.id === 'lonely-spec');
      assert.ok(spec, 'Should parse the spec even without assertions/');

      const specAssertions = assertions.filter(a => a.parent === 'lonely-spec');
      assert.equal(specAssertions.length, 0, 'Should have zero assertions');
    } finally {
      if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
    }
  });
});
