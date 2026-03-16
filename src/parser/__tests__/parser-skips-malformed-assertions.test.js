import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'fs';
import path from 'path';
import { parseAllSpecs } from '../index.js';

describe('Parser skips malformed assertion files', () => {

  function createTempDir(name) {
    const dir = path.join(process.cwd(), `temp-${name}-${Date.now()}`);
    fs.mkdirSync(dir, { recursive: true });
    return dir;
  }

  function cleanupTempDir(dir) {
    if (fs.existsSync(dir)) fs.rmSync(dir, { recursive: true });
  }

  function writeSpec(tempDir, specId, extraFields = '') {
    const specDir = path.join(tempDir, specId);
    const assertionsDir = path.join(specDir, 'assertions');
    fs.mkdirSync(assertionsDir, { recursive: true });
    fs.writeFileSync(path.join(specDir, `${specId}.md`), `---
id: ${specId}
created: 2026-01-28T21:35:00Z
priority: 1
${extraFields}
---
# ${specId}

A test spec.`);
    return assertionsDir;
  }

  function writeAssertion(assertionsDir, id, parent, extra = '') {
    fs.writeFileSync(path.join(assertionsDir, `${id}.md`), `---
id: ${id}
parent: ${parent}
created: 2026-01-28T21:35:00Z
priority: 1
status: not_started
${extra}
---
# ${id}

A test assertion.`);
  }

  test('assertion files missing required fields are skipped with a stderr warning', () => {
    const tempDir = createTempDir('missing-fields');
    try {
      const assertionsDir = writeSpec(tempDir, 'test-spec');

      // Valid assertion
      writeAssertion(assertionsDir, 'valid-assertion', 'test-spec');

      // Assertion missing 'parent' field (required)
      fs.writeFileSync(path.join(assertionsDir, 'missing-parent.md'), `---
id: missing-parent
created: 2026-01-28T21:35:00Z
priority: 1
status: not_started
---
# Missing Parent

This assertion is missing the required parent field.`);

      // Assertion missing 'id' field (required)
      fs.writeFileSync(path.join(assertionsDir, 'missing-id.md'), `---
parent: test-spec
created: 2026-01-28T21:35:00Z
priority: 1
status: not_started
---
# Missing ID

This assertion is missing the required id field.`);

      // Assertion missing 'priority' field (required)
      fs.writeFileSync(path.join(assertionsDir, 'missing-priority.md'), `---
id: missing-priority
parent: test-spec
created: 2026-01-28T21:35:00Z
status: not_started
---
# Missing Priority

This assertion is missing the required priority field.`);

      const { specs, assertions } = parseAllSpecs(tempDir);

      // Valid assertion should be parsed
      assert.ok(
        assertions.find(a => a.id === 'valid-assertion'),
        'Should parse the valid assertion'
      );

      // Malformed assertions should be skipped
      assert.equal(
        assertions.find(a => a.id === 'missing-parent'),
        undefined,
        'Should skip assertion missing parent field'
      );
      assert.equal(
        assertions.find(a => a.id === 'missing-priority'),
        undefined,
        'Should skip assertion missing priority field'
      );

      // The spec should still be parsed
      assert.ok(specs.find(s => s.id === 'test-spec'), 'Spec should still be parsed');
    } finally {
      cleanupTempDir(tempDir);
    }
  });

  test('assertion files with unparseable YAML frontmatter are skipped with a stderr warning', () => {
    const tempDir = createTempDir('bad-yaml');
    try {
      const assertionsDir = writeSpec(tempDir, 'test-spec');

      // Valid assertion
      writeAssertion(assertionsDir, 'valid-assertion', 'test-spec');

      // Assertion with no closing --- delimiter
      fs.writeFileSync(path.join(assertionsDir, 'no-closing-delimiter.md'), `---
id: no-closing-delimiter
parent: test-spec
created: 2026-01-28T21:35:00Z
priority: 1
status: not_started

# No Closing Delimiter

This file has no closing frontmatter delimiter.`);

      // Assertion with garbage content that starts with ---
      fs.writeFileSync(path.join(assertionsDir, 'garbage-yaml.md'), `---
this is: [not: valid: yaml: {{{}}}
---
# Garbage YAML`);

      const { specs, assertions } = parseAllSpecs(tempDir);

      assert.ok(
        assertions.find(a => a.id === 'valid-assertion'),
        'Should parse the valid assertion'
      );
      assert.equal(
        assertions.find(a => a.id === 'no-closing-delimiter'),
        undefined,
        'Should skip assertion with no closing delimiter'
      );
      assert.equal(assertions.length, 1, 'Only one assertion should be parsed');
    } finally {
      cleanupTempDir(tempDir);
    }
  });

  test('parent spec computed status is derived only from successfully-parsed assertions', () => {
    const tempDir = createTempDir('computed-status');
    try {
      const assertionsDir = writeSpec(tempDir, 'test-spec');

      // One valid assertion that is done
      fs.writeFileSync(path.join(assertionsDir, 'done-assertion.md'), `---
id: done-assertion
parent: test-spec
created: 2026-01-28T21:35:00Z
priority: 1
status: done
---
# Done Assertion

This assertion is done.`);

      // One malformed assertion (missing parent) - should be skipped
      fs.writeFileSync(path.join(assertionsDir, 'malformed-assertion.md'), `---
id: malformed-assertion
created: 2026-01-28T21:35:00Z
priority: 1
status: not_started
---
# Malformed

Missing parent field.`);

      const { specs, assertions } = parseAllSpecs(tempDir);

      const spec = specs.find(s => s.id === 'test-spec');
      assert.ok(spec, 'Spec should exist');

      // Only the done assertion should be parsed
      assert.equal(assertions.length, 1, 'Only one assertion should be parsed');
      assert.equal(assertions[0].id, 'done-assertion');

      // Status should be 'done' since the only valid assertion is done
      assert.equal(spec.status, 'done',
        'Parent spec status should be done (only valid assertion is done)');
    } finally {
      cleanupTempDir(tempDir);
    }
  });

  test('a spec with zero parseable assertions (all malformed) is treated as having no assertions', () => {
    const tempDir = createTempDir('all-malformed');
    try {
      const assertionsDir = writeSpec(tempDir, 'test-spec');

      // All assertions are malformed
      fs.writeFileSync(path.join(assertionsDir, 'malformed-one.md'), `---
id: malformed-one
created: 2026-01-28T21:35:00Z
priority: 1
status: not_started
---
# Malformed One

Missing parent field.`);

      fs.writeFileSync(path.join(assertionsDir, 'malformed-two.md'), `---
id: malformed-two
created: invalid-date
priority: 1
parent: test-spec
status: not_started
---
# Malformed Two

Invalid date format.`);

      const { specs, assertions } = parseAllSpecs(tempDir);

      const spec = specs.find(s => s.id === 'test-spec');
      assert.ok(spec, 'Spec should still exist');

      // No assertions should be parsed
      const specAssertions = assertions.filter(a => a.parent === 'test-spec');
      assert.equal(specAssertions.length, 0,
        'All malformed assertions should be skipped');

      // Spec status should be not_started (same as zero assertions)
      assert.equal(spec.status, 'not_started',
        'Spec with all malformed assertions should have not_started status');
    } finally {
      cleanupTempDir(tempDir);
    }
  });

});
