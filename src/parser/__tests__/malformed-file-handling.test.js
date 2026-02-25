import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'fs';
import path from 'path';
import { parseAllSpecs } from '../index.js';

describe('Malformed File Handling', () => {
  test('parser skips malformed spec files and continues processing valid ones', () => {
    const tempDir = path.join(process.cwd(), 'temp-malformed-spec-test');
    const validSpecDir = path.join(tempDir, 'valid-spec');
    const malformedSpecDir = path.join(tempDir, 'malformed-spec');
    const validAssertionsDir = path.join(validSpecDir, 'assertions');
    const malformedAssertionsDir = path.join(malformedSpecDir, 'assertions');

    try {
      fs.mkdirSync(validAssertionsDir, { recursive: true });
      fs.mkdirSync(malformedAssertionsDir, { recursive: true });

      fs.writeFileSync(join(validSpecDir, 'valid-spec.md'), `---
id: valid-spec
created: 2026-01-28T21:35:00Z
priority: 1
---
# Valid Spec

This spec should be parsed successfully.`);

      fs.writeFileSync(join(validAssertionsDir, 'valid-assertion.md'), `---
id: valid-assertion
parent: valid-spec
created: 2026-01-28T21:35:00Z
priority: 1
status: not_started
---
# Valid Assertion

This assertion should be parsed successfully.`);

      fs.writeFileSync(join(malformedSpecDir, 'malformed-spec.md'), `---
id: malformed-spec
created: invalid-date-format
priority: "not-a-number"
---
# Malformed Spec

This spec has invalid YAML frontmatter.`);

      const originalValidSpecPath = path.join(process.cwd(), 'specs', 'valid-spec');
      const originalMalformedSpecPath = path.join(process.cwd(), 'specs', 'malformed-spec');
      fs.symlinkSync(validSpecDir, originalValidSpecPath);
      fs.symlinkSync(malformedSpecDir, originalMalformedSpecPath);

      try {
        const { specs, assertions } = parseAllSpecs();

        assert.ok(specs.find(s => s.id === 'valid-spec'), 'Should parse valid spec');
        assert.ok(assertions.find(a => a.id === 'valid-assertion'), 'Should parse valid assertion');
        assert.equal(specs.find(s => s.id === 'malformed-spec'), undefined, 'Should skip malformed spec');

      } finally {
        fs.unlinkSync(originalValidSpecPath);
        fs.unlinkSync(originalMalformedSpecPath);
      }

    } finally {
      if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
    }
  });

  test('parser continues processing when some assertion files are malformed', () => {
    const tempDir = path.join(process.cwd(), 'temp-malformed-assertion-test');
    const assertionsDir = path.join(tempDir, 'assertions');

    try {
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(tempDir, 'test-malformed-assertion.md'), `---
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

      const originalSpecsPath = path.join(process.cwd(), 'specs', 'test-malformed-assertion');
      fs.symlinkSync(tempDir, originalSpecsPath);

      try {
        const { specs, assertions } = parseAllSpecs();

        assert.ok(specs.find(s => s.id === 'test-malformed-assertion'), 'Should parse spec');
        assert.ok(assertions.find(a => a.id === 'valid-assertion'), 'Should parse valid assertion');
        assert.equal(assertions.find(a => a.id === 'malformed-assertion'), undefined, 'Should skip malformed assertion');

      } finally {
        fs.unlinkSync(originalSpecsPath);
      }

    } finally {
      if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
    }
  });
});

function join(...args) {
  return path.join(...args);
}
