import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'fs';
import path from 'path';
import { parseAllSpecs, parseFrontmatter, validateDependsOn, detectCircularDependencies } from '../index.js';

// Helper to cleanup temp directories and symlinks
function cleanupTest(tempDir, symlinkPath) {
  try {
    if (symlinkPath && fs.existsSync(symlinkPath)) {
      fs.unlinkSync(symlinkPath);
    }
  } catch (err) {
    // Ignore cleanup errors
  }
  try {
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true });
    }
  } catch (err) {
    // Ignore cleanup errors
  }
}

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

  test('accepts omitted depends-on field', () => {
    const tempDir = path.join(process.cwd(), 'temp-depends-omitted-test');
    const assertionsDir = path.join(tempDir, 'assertions');
    const symlinkPath = path.join(process.cwd(), 'specs', 'temp-depends-omitted-test');

    try {
      cleanupTest(tempDir, symlinkPath);
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(tempDir, 'temp-depends-omitted-test.md'), `---
id: temp-depends-omitted-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Omitted Test Spec`);

      fs.writeFileSync(path.join(assertionsDir, 'test-assertion.md'), `---
id: test-assertion
parent: temp-depends-omitted-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Test Assertion`);

      fs.symlinkSync(tempDir, symlinkPath);

      // Should not throw
      const { assertions } = parseAllSpecs();
      const testAssertion = assertions.find(a => a.id === 'test-assertion');
      assert.ok(testAssertion, 'Should parse assertion without depends-on field');
      assert.equal(testAssertion.dependsOn, undefined, 'Should have undefined dependsOn when omitted');
    } finally {
      cleanupTest(tempDir, symlinkPath);
    }
  });

  test('rejects invalid type for depends-on field', () => {
    const tempDir = path.join(process.cwd(), 'temp-depends-invalid-type-test');
    const assertionsDir = path.join(tempDir, 'assertions');
    const symlinkPath = path.join(process.cwd(), 'specs', 'temp-depends-invalid-type-test');

    try {
      cleanupTest(tempDir, symlinkPath);
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(tempDir, 'temp-depends-invalid-type-test.md'), `---
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

      fs.symlinkSync(tempDir, symlinkPath);

      assert.throws(
        () => parseAllSpecs(),
        /Field 'depends-on' must be a string or null/,
        'Should reject array type for depends-on'
      );
    } finally {
      cleanupTest(tempDir, symlinkPath);
    }
  });

  test('rejects invalid format for depends-on field', () => {
    const tempDir = path.join(process.cwd(), 'temp-depends-invalid-format-test');
    const assertionsDir = path.join(tempDir, 'assertions');
    const symlinkPath = path.join(process.cwd(), 'specs', 'temp-depends-invalid-format-test');

    try {
      cleanupTest(tempDir, symlinkPath);
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(tempDir, 'temp-depends-invalid-format-test.md'), `---
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

      fs.symlinkSync(tempDir, symlinkPath);

      assert.throws(
        () => parseAllSpecs(),
        /Field 'depends-on' must be kebab-case/,
        'Should reject non-kebab-case format for depends-on'
      );
    } finally {
      cleanupTest(tempDir, symlinkPath);
    }
  });

  test('rejects non-existent assertion reference', () => {
    const tempDir = path.join(process.cwd(), 'temp-depends-nonexistent-test');
    const assertionsDir = path.join(tempDir, 'assertions');
    const symlinkPath = path.join(process.cwd(), 'specs', 'temp-depends-nonexistent-test');

    try {
      cleanupTest(tempDir, symlinkPath);
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(tempDir, 'temp-depends-nonexistent-test.md'), `---
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

      fs.symlinkSync(tempDir, symlinkPath);

      assert.throws(
        () => parseAllSpecs(),
        /Field 'depends-on' references non-existent assertion 'nonexistent-assertion'/,
        'Should reject reference to non-existent assertion'
      );
    } finally {
      cleanupTest(tempDir, symlinkPath);
    }
  });

  test('rejects self-reference in depends-on', () => {
    const tempDir = path.join(process.cwd(), 'temp-depends-self-test');
    const assertionsDir = path.join(tempDir, 'assertions');
    const symlinkPath = path.join(process.cwd(), 'specs', 'temp-depends-self-test');

    try {
      cleanupTest(tempDir, symlinkPath);
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(tempDir, 'temp-depends-self-test.md'), `---
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

      fs.symlinkSync(tempDir, symlinkPath);

      assert.throws(
        () => parseAllSpecs(),
        /Field 'depends-on' cannot reference itself/,
        'Should reject self-reference in depends-on'
      );
    } finally {
      cleanupTest(tempDir, symlinkPath);
    }
  });

  test('detects circular dependencies', () => {
    const tempDir = path.join(process.cwd(), 'temp-depends-circular-test');
    const assertionsDir = path.join(tempDir, 'assertions');
    const symlinkPath = path.join(process.cwd(), 'specs', 'temp-depends-circular-test');

    try {
      cleanupTest(tempDir, symlinkPath);
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(tempDir, 'temp-depends-circular-test.md'), `---
id: temp-depends-circular-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Circular Test Spec`);

      // Create circular dependency: a → b → c → a
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

      fs.symlinkSync(tempDir, symlinkPath);

      assert.throws(
        () => parseAllSpecs(),
        /Circular dependency detected/,
        'Should detect circular dependencies'
      );
    } finally {
      cleanupTest(tempDir, symlinkPath);
    }
  });

  test('accepts valid dependency chain without cycles', () => {
    const tempDir = path.join(process.cwd(), 'temp-depends-valid-chain-test');
    const assertionsDir = path.join(tempDir, 'assertions');
    const symlinkPath = path.join(process.cwd(), 'specs', 'temp-depends-valid-chain-test');

    try {
      cleanupTest(tempDir, symlinkPath);
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(tempDir, 'temp-depends-valid-chain-test.md'), `---
id: temp-depends-valid-chain-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Valid Chain Test Spec`);

      // Create valid chain: a → b → c
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

      fs.symlinkSync(tempDir, symlinkPath);

      // Should not throw
      const { assertions } = parseAllSpecs();
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
      cleanupTest(tempDir, symlinkPath);
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
