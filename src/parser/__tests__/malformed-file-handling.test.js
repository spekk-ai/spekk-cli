import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'fs';
import path from 'path';
import os from 'os';
import { parseAllSpecs } from '../index.js';

describe('Malformed File Handling', () => {
  test('parser skips malformed spec files and continues processing valid ones', () => {
    const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-malformed-'));
    const specsDir = path.join(testDir, 'specs');
    const validSpecDir = path.join(specsDir, 'valid-spec');
    const malformedSpecDir = path.join(specsDir, 'malformed-spec');
    const validAssertionsDir = path.join(validSpecDir, 'assertions');
    const malformedAssertionsDir = path.join(malformedSpecDir, 'assertions');

    try {
      fs.mkdirSync(validAssertionsDir, { recursive: true });
      fs.mkdirSync(malformedAssertionsDir, { recursive: true });

      fs.writeFileSync(path.join(validSpecDir, 'valid-spec.md'), `---
id: valid-spec
created: 2026-01-28T21:35:00Z
priority: 1
---
# Valid Spec

This spec should be parsed successfully.`);

      fs.writeFileSync(path.join(validAssertionsDir, 'valid-assertion.md'), `---
id: valid-assertion
parent: valid-spec
created: 2026-01-28T21:35:00Z
priority: 1
status: not_started
---
# Valid Assertion

This assertion should be parsed successfully.`);

      fs.writeFileSync(path.join(malformedSpecDir, 'malformed-spec.md'), `---
id: malformed-spec
created: invalid-date-format
priority: "not-a-number"
---
# Malformed Spec

This spec has invalid YAML frontmatter.`);

      const { specs, assertions } = parseAllSpecs(specsDir);

      assert.ok(specs.find(s => s.id === 'valid-spec'), 'Should parse valid spec');
      assert.ok(assertions.find(a => a.id === 'valid-assertion'), 'Should parse valid assertion');
      assert.equal(specs.find(s => s.id === 'malformed-spec'), undefined, 'Should skip malformed spec');
    } finally {
      if (fs.existsSync(testDir)) fs.rmSync(testDir, { recursive: true, force: true });
    }
  });

  test('parser continues processing when some assertion files are malformed', () => {
    const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-malformed-assertion-'));
    const specsDir = path.join(testDir, 'specs');
    const specDir = path.join(specsDir, 'test-malformed-assertion');
    const assertionsDir = path.join(specDir, 'assertions');

    try {
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(specDir, 'test-malformed-assertion.md'), `---
id: test-malformed-assertion
created: 2026-01-28T21:35:00Z
priority: 1
---
# Test Spec

Parent spec for testing malformed assertions.`);

      fs.writeFileSync(path.join(assertionsDir, 'valid-assertion.md'), `---
id: valid-assertion
parent: test-malformed-assertion
created: 2026-01-28T21:35:00Z
priority: 1
status: not_started
---
# Valid Assertion

This assertion should be parsed successfully.`);

      fs.writeFileSync(path.join(assertionsDir, 'malformed-assertion.md'), `---
id: malformed-assertion
created: 2026-01-28T21:35:00Z
priority: 1
status: not_started
---
# Malformed Assertion

This assertion is missing the required parent field.`);

      const { specs, assertions } = parseAllSpecs(specsDir);

      assert.ok(specs.find(s => s.id === 'test-malformed-assertion'), 'Should parse spec');
      assert.ok(assertions.find(a => a.id === 'valid-assertion'), 'Should parse valid assertion');
      assert.equal(assertions.find(a => a.id === 'malformed-assertion'), undefined, 'Should skip malformed assertion');
    } finally {
      if (fs.existsSync(testDir)) fs.rmSync(testDir, { recursive: true, force: true });
    }
  });
});
