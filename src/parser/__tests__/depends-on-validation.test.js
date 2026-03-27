import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'fs';
import path from 'path';
import os from 'os';
import { parseAllSpecs, parseFrontmatter, validateDependsOn, detectCircularDependencies } from '../index.js';

describe('depends-on Field Validation', () => {
  test('parses depends-on field and converts to camelCase', () => {
    const yaml = `---
id: test-assertion
parent: test-spec
created: 2026-01-20T16:00:00Z
priority: 1
depends-on: other-assertion
---

# Test`;

    const { data } = parseFrontmatter(yaml);
    assert.equal(data.dependsOn, 'other-assertion', 'Should convert depends-on to dependsOn');
    assert.equal(data['depends-on'], undefined, 'Should not have kebab-case key in parsed data');
  });

  test('rejects invalid type for depends-on field', () => {
    const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-depends-type-'));
    const specsDir = path.join(testDir, 'specs');
    const specDir = path.join(specsDir, 'temp-depends-invalid-type-test');
    const assertionsDir = path.join(specDir, 'assertions');

    try {
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(specDir, 'temp-depends-invalid-type-test.md'), `---
id: temp-depends-invalid-type-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Invalid Type Test Spec`);

      // Create assertion with array instead of string
      fs.writeFileSync(path.join(assertionsDir, 'test-assertion.md'), `---
id: test-assertion
parent: temp-depends-invalid-type-test
created: 2026-01-20T16:00:00Z
priority: 1
depends-on:
  - other-assertion
  - another-assertion
---

# Test Assertion`);

      assert.throws(
        () => parseAllSpecs(specsDir),
        /Field 'depends-on' must be a string or null/,
        'Should reject array type for depends-on'
      );
    } finally {
      if (fs.existsSync(testDir)) fs.rmSync(testDir, { recursive: true, force: true });
    }
  });

  test('rejects invalid format for depends-on field', () => {
    const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-depends-format-'));
    const specsDir = path.join(testDir, 'specs');
    const specDir = path.join(specsDir, 'temp-depends-invalid-format-test');
    const assertionsDir = path.join(specDir, 'assertions');

    try {
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(specDir, 'temp-depends-invalid-format-test.md'), `---
id: temp-depends-invalid-format-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Invalid Format Test Spec`);

      fs.writeFileSync(path.join(assertionsDir, 'other-assertion.md'), `---
id: other-assertion
parent: temp-depends-invalid-format-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Other Assertion`);

      // Create assertion with camelCase instead of kebab-case
      fs.writeFileSync(path.join(assertionsDir, 'test-assertion.md'), `---
id: test-assertion
parent: temp-depends-invalid-format-test
created: 2026-01-20T16:00:00Z
priority: 1
depends-on: otherAssertion
---

# Test Assertion`);

      assert.throws(
        () => parseAllSpecs(specsDir),
        /Field 'depends-on' must be kebab-case/,
        'Should reject non-kebab-case format for depends-on'
      );
    } finally {
      if (fs.existsSync(testDir)) fs.rmSync(testDir, { recursive: true, force: true });
    }
  });

  test('rejects non-existent assertion reference', () => {
    const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-depends-nonexist-'));
    const specsDir = path.join(testDir, 'specs');
    const specDir = path.join(specsDir, 'temp-depends-nonexistent-test');
    const assertionsDir = path.join(specDir, 'assertions');

    try {
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(specDir, 'temp-depends-nonexistent-test.md'), `---
id: temp-depends-nonexistent-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Nonexistent Test Spec`);

      fs.writeFileSync(path.join(assertionsDir, 'test-assertion.md'), `---
id: test-assertion
parent: temp-depends-nonexistent-test
created: 2026-01-20T16:00:00Z
priority: 1
depends-on: nonexistent-assertion
---

# Test Assertion`);

      assert.throws(
        () => parseAllSpecs(specsDir),
        /Field 'depends-on' references non-existent assertion 'nonexistent-assertion'/,
        'Should reject reference to non-existent assertion'
      );
    } finally {
      if (fs.existsSync(testDir)) fs.rmSync(testDir, { recursive: true, force: true });
    }
  });

  test('rejects self-reference in depends-on', () => {
    const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-depends-self-'));
    const specsDir = path.join(testDir, 'specs');
    const specDir = path.join(specsDir, 'temp-depends-self-test');
    const assertionsDir = path.join(specDir, 'assertions');

    try {
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(specDir, 'temp-depends-self-test.md'), `---
id: temp-depends-self-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Self Test Spec`);

      fs.writeFileSync(path.join(assertionsDir, 'test-assertion.md'), `---
id: test-assertion
parent: temp-depends-self-test
created: 2026-01-20T16:00:00Z
priority: 1
depends-on: test-assertion
---

# Test Assertion`);

      assert.throws(
        () => parseAllSpecs(specsDir),
        /Field 'depends-on' cannot reference itself/,
        'Should reject self-reference in depends-on'
      );
    } finally {
      if (fs.existsSync(testDir)) fs.rmSync(testDir, { recursive: true, force: true });
    }
  });

  test('detects circular dependencies', () => {
    const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-depends-circular-'));
    const specsDir = path.join(testDir, 'specs');
    const specDir = path.join(specsDir, 'temp-depends-circular-test');
    const assertionsDir = path.join(specDir, 'assertions');

    try {
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(specDir, 'temp-depends-circular-test.md'), `---
id: temp-depends-circular-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Circular Test Spec`);

      // Create circular dependency: a -> b -> c -> a
      fs.writeFileSync(path.join(assertionsDir, 'assertion-a.md'), `---
id: assertion-a
parent: temp-depends-circular-test
created: 2026-01-20T16:00:00Z
priority: 1
depends-on: assertion-b
---

# Assertion A`);

      fs.writeFileSync(path.join(assertionsDir, 'assertion-b.md'), `---
id: assertion-b
parent: temp-depends-circular-test
created: 2026-01-20T16:00:00Z
priority: 1
depends-on: assertion-c
---

# Assertion B`);

      fs.writeFileSync(path.join(assertionsDir, 'assertion-c.md'), `---
id: assertion-c
parent: temp-depends-circular-test
created: 2026-01-20T16:00:00Z
priority: 1
depends-on: assertion-a
---

# Assertion C`);

      assert.throws(
        () => parseAllSpecs(specsDir),
        /Circular dependency detected/,
        'Should detect circular dependencies'
      );
    } finally {
      if (fs.existsSync(testDir)) fs.rmSync(testDir, { recursive: true, force: true });
    }
  });

  test('accepts valid dependency chain without cycles', () => {
    const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-depends-valid-'));
    const specsDir = path.join(testDir, 'specs');
    const specDir = path.join(specsDir, 'temp-depends-valid-chain-test');
    const assertionsDir = path.join(specDir, 'assertions');

    try {
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(specDir, 'temp-depends-valid-chain-test.md'), `---
id: temp-depends-valid-chain-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Valid Chain Test Spec`);

      // Create valid chain: a -> b -> c
      fs.writeFileSync(path.join(assertionsDir, 'assertion-c.md'), `---
id: assertion-c
parent: temp-depends-valid-chain-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Assertion C`);

      fs.writeFileSync(path.join(assertionsDir, 'assertion-b.md'), `---
id: assertion-b
parent: temp-depends-valid-chain-test
created: 2026-01-20T16:00:00Z
priority: 1
depends-on: assertion-c
---

# Assertion B`);

      fs.writeFileSync(path.join(assertionsDir, 'assertion-a.md'), `---
id: assertion-a
parent: temp-depends-valid-chain-test
created: 2026-01-20T16:00:00Z
priority: 1
depends-on: assertion-b
---

# Assertion A`);

      // Should not throw
      const { assertions } = parseAllSpecs(specsDir);
      const assertionA = assertions.find(a => a.id === 'assertion-a');
      const assertionB = assertions.find(a => a.id === 'assertion-b');
      const assertionC = assertions.find(a => a.id === 'assertion-c');

      assert.ok(assertionA, 'Should parse assertion A');
      assert.ok(assertionB, 'Should parse assertion B');
      assert.ok(assertionC, 'Should parse assertion C');
      assert.equal(assertionA.dependsOn, 'assertion-b', 'A should depend on B');
      assert.equal(assertionB.dependsOn, 'assertion-c', 'B should depend on C');
      assert.equal(assertionC.dependsOn, undefined, 'C should have no dependency');
    } finally {
      if (fs.existsSync(testDir)) fs.rmSync(testDir, { recursive: true, force: true });
    }
  });

  test('unit test: validateDependsOn rejects invalid types', () => {
    const testData = {
      id: 'test-assertion',
      dependsOn: ['invalid', 'array']
    };
    const allAssertions = [{ id: 'test-assertion' }];

    assert.throws(
      () => validateDependsOn(testData, 'test.md', allAssertions),
      /Field 'depends-on' must be a string or null/,
      'Should reject array in unit test'
    );
  });

  test('unit test: detectCircularDependencies detects simple cycle', () => {
    const assertions = [
      { id: 'a', dependsOn: 'b' },
      { id: 'b', dependsOn: 'a' }
    ];

    assert.throws(
      () => detectCircularDependencies(assertions),
      /Circular dependency detected/,
      'Should detect simple cycle in unit test'
    );
  });
});
