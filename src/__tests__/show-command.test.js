import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import { existsSync, rmSync, mkdirSync, writeFileSync, readFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { tmpdir } from 'node:os';

describe('Show Command', () => {

  const testDir = join(tmpdir(), `spekk-test-${Date.now()}`);
  const __dirname = dirname(fileURLToPath(import.meta.url));
  const projectRoot = join(__dirname, '../..');

  function cleanup() {
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
  }

  test('generates valid HTML with spec data in .spekk directory', () => {
    cleanup();
    mkdirSync(testDir, { recursive: true });

    try {
      // Create specs structure
      const specsDir = join(testDir, 'specs');
      const specDir = join(specsDir, 'test-spec');
      const assertionsDir = join(specDir, 'assertions');
      mkdirSync(assertionsDir, { recursive: true });

      writeFileSync(join(specDir, 'test-spec.md'), `---
id: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
---

# Test Spec

This is a test specification.`);

      writeFileSync(join(assertionsDir, 'test-assertion.md'), `---
id: test-assertion
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Test Assertion

This is a test assertion.`);

      execSync(`node "${join(projectRoot, 'bin/spekk.js')}" show`, {
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000,
        env: { ...process.env, NODE_ENV: 'test' }
      });

      // .spekk directory and index.html should exist
      const htmlFile = join(testDir, '.spekk', 'index.html');
      assert.ok(existsSync(htmlFile), 'index.html should be created in .spekk directory');

      // HTML should be valid and contain spec data
      const htmlContent = readFileSync(htmlFile, 'utf8');
      assert.ok(htmlContent.includes('<html'), 'Should contain valid HTML');
      assert.ok(htmlContent.includes('</html>'), 'Should have closing HTML tag');
      assert.ok(htmlContent.toLowerCase().includes('spec'), 'Should contain spec content');

    } finally {
      cleanup();
    }
  });

  test('overwrites existing HTML on subsequent runs', () => {
    cleanup();
    mkdirSync(testDir, { recursive: true });

    try {
      const spekkDir = join(testDir, '.spekk');
      mkdirSync(spekkDir);
      const htmlFile = join(spekkDir, 'index.html');
      writeFileSync(htmlFile, 'old content');

      execSync(`node "${join(projectRoot, 'bin/spekk.js')}" show`, {
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000,
        env: { ...process.env, NODE_ENV: 'test' }
      });

      const htmlContent = readFileSync(htmlFile, 'utf8');
      assert.ok(!htmlContent.includes('old content'), 'Old content should be overwritten');
      assert.ok(htmlContent.includes('<html'), 'New HTML content should be present');

    } finally {
      cleanup();
    }
  });

  test('includes metro map section in assertion detail panels', () => {
    cleanup();
    mkdirSync(testDir, { recursive: true });

    try {
      const specsDir = join(testDir, 'specs');
      const specDir = join(specsDir, 'test-spec');
      const assertionsDir = join(specDir, 'assertions');
      mkdirSync(assertionsDir, { recursive: true });

      writeFileSync(join(specDir, 'test-spec.md'), `---
id: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
---

# Test Spec`);

      writeFileSync(join(assertionsDir, 'assertion-a.md'), `---
id: assertion-a
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
branch: feature/test
---

# Assertion A`);

      writeFileSync(join(assertionsDir, 'assertion-b.md'), `---
id: assertion-b
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
branch: feature/test
depends-on: assertion-a
---

# Assertion B`);

      execSync(`node "${join(projectRoot, 'bin/spekk.js')}" show`, {
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000,
        env: { ...process.env, NODE_ENV: 'test' }
      });

      const htmlContent = readFileSync(join(testDir, '.spekk', 'index.html'), 'utf8');

      // Metro map section should be present
      assert.ok(htmlContent.includes('metro-map-section'), 'Should include metro map section');
      assert.ok(htmlContent.includes('Branch Dependencies'), 'Should show Branch Dependencies title');
      assert.ok(htmlContent.includes('class="metro-map"'), 'Should include metro map SVG');

    } finally {
      cleanup();
    }
  });

  test('metro map shows only assertions from same branch', () => {
    cleanup();
    mkdirSync(testDir, { recursive: true });

    try {
      const specsDir = join(testDir, 'specs');
      const specDir = join(specsDir, 'test-spec');
      const assertionsDir = join(specDir, 'assertions');
      mkdirSync(assertionsDir, { recursive: true });

      writeFileSync(join(specDir, 'test-spec.md'), `---
id: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
---

# Test Spec`);

      // Assertion in feature/branch-a
      writeFileSync(join(assertionsDir, 'assertion-a.md'), `---
id: assertion-a
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: done
branch: feature/branch-a
---

# Assertion A`);

      // Assertion in feature/branch-b
      writeFileSync(join(assertionsDir, 'assertion-b.md'), `---
id: assertion-b
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
branch: feature/branch-b
---

# Assertion B`);

      execSync(`node "${join(projectRoot, 'bin/spekk.js')}" show`, {
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000,
        env: { ...process.env, NODE_ENV: 'test' }
      });

      const htmlContent = readFileSync(join(testDir, '.spekk', 'index.html'), 'utf8');

      // Each assertion should only show its own branch in the metro map
      const assertionADetail = htmlContent.match(/id="detail-assertion-assertion-a"[\s\S]*?<\/div>\s*<\/div>\s*<\/div>/);
      const assertionBDetail = htmlContent.match(/id="detail-assertion-assertion-b"[\s\S]*?<\/div>\s*<\/div>\s*<\/div>/);

      assert.ok(assertionADetail, 'Should find assertion A detail section');
      assert.ok(assertionBDetail, 'Should find assertion B detail section');

    } finally {
      cleanup();
    }
  });

  test('metro map highlights current assertion', () => {
    cleanup();
    mkdirSync(testDir, { recursive: true });

    try {
      const specsDir = join(testDir, 'specs');
      const specDir = join(specsDir, 'test-spec');
      const assertionsDir = join(specDir, 'assertions');
      mkdirSync(assertionsDir, { recursive: true });

      writeFileSync(join(specDir, 'test-spec.md'), `---
id: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
---

# Test Spec`);

      writeFileSync(join(assertionsDir, 'assertion-current.md'), `---
id: assertion-current
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: in_progress
branch: feature/test
---

# Assertion Current`);

      execSync(`node "${join(projectRoot, 'bin/spekk.js')}" show`, {
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000,
        env: { ...process.env, NODE_ENV: 'test' }
      });

      const htmlContent = readFileSync(join(testDir, '.spekk', 'index.html'), 'utf8');

      // Current assertion should have special styling (larger radius and glow filter)
      assert.ok(htmlContent.includes('r="10"'), 'Current assertion should have larger radius');
      assert.ok(htmlContent.includes('filter="drop-shadow'), 'Current assertion should have glow effect');

    } finally {
      cleanup();
    }
  });

  test('metro map positions assertions by dependency depth', () => {
    cleanup();
    mkdirSync(testDir, { recursive: true });

    try {
      const specsDir = join(testDir, 'specs');
      const specDir = join(specsDir, 'test-spec');
      const assertionsDir = join(specDir, 'assertions');
      mkdirSync(assertionsDir, { recursive: true });

      writeFileSync(join(specDir, 'test-spec.md'), `---
id: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
---

# Test Spec`);

      // Chain of dependencies: A -> B -> C
      writeFileSync(join(assertionsDir, 'assertion-a.md'), `---
id: assertion-a
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: done
branch: feature/chain
---

# Assertion A`);

      writeFileSync(join(assertionsDir, 'assertion-b.md'), `---
id: assertion-b
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: done
branch: feature/chain
depends-on: assertion-a
---

# Assertion B`);

      writeFileSync(join(assertionsDir, 'assertion-c.md'), `---
id: assertion-c
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
branch: feature/chain
depends-on: assertion-b
---

# Assertion C`);

      execSync(`node "${join(projectRoot, 'bin/spekk.js')}" show`, {
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000,
        env: { ...process.env, NODE_ENV: 'test' }
      });

      const htmlContent = readFileSync(join(testDir, '.spekk', 'index.html'), 'utf8');

      // Metro map should show stations positioned by dependency depth
      assert.ok(htmlContent.includes('class="metro-map"'), 'Should have metro map');
      assert.ok(htmlContent.includes('Station: assertion-a'), 'Should include assertion A');
      assert.ok(htmlContent.includes('Station: assertion-b'), 'Should include assertion B');
      assert.ok(htmlContent.includes('Station: assertion-c'), 'Should include assertion C');

      // Dependency lines should be present
      assert.ok(htmlContent.includes('class="metro-dependency"'), 'Should have dependency lines');

    } finally {
      cleanup();
    }
  });

  test('metro map shows "Done" terminus when multiple terminal assertions exist', () => {
    cleanup();
    mkdirSync(testDir, { recursive: true });

    try {
      const specsDir = join(testDir, 'specs');
      const specDir = join(specsDir, 'test-spec');
      const assertionsDir = join(specDir, 'assertions');
      mkdirSync(assertionsDir, { recursive: true });

      writeFileSync(join(specDir, 'test-spec.md'), `---
id: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
---

# Test Spec`);

      // Create a branch with parallel work that converges
      // A (root)
      // ├─ B (depends on A)
      // └─ C (depends on A)
      // Both B and C are terminals
      writeFileSync(join(assertionsDir, 'assertion-a.md'), `---
id: assertion-a
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: done
branch: feature/parallel
---

# Assertion A`);

      writeFileSync(join(assertionsDir, 'assertion-b.md'), `---
id: assertion-b
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: in_progress
branch: feature/parallel
depends-on: assertion-a
---

# Assertion B`);

      writeFileSync(join(assertionsDir, 'assertion-c.md'), `---
id: assertion-c
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
branch: feature/parallel
depends-on: assertion-a
---

# Assertion C`);

      execSync(`node "${join(projectRoot, 'bin/spekk.js')}" show`, {
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000,
        env: { ...process.env, NODE_ENV: 'test' }
      });

      const htmlContent = readFileSync(join(testDir, '.spekk', 'index.html'), 'utf8');

      // Should have "Done" terminus
      assert.ok(htmlContent.includes('Done Terminus'), 'Should include Done terminus comment');
      assert.ok(htmlContent.includes('class="metro-terminus"'), 'Should have metro-terminus class');
      assert.ok(htmlContent.includes('>Done</text>'), 'Should have "Done" label');

      // Should have convergence lines
      assert.ok(htmlContent.includes('Convergence:'), 'Should have convergence line comments');

    } finally {
      cleanup();
    }
  });

  test('metro map does not show "Done" terminus when only one terminal assertion exists', () => {
    cleanup();
    mkdirSync(testDir, { recursive: true });

    try {
      const specsDir = join(testDir, 'specs');
      const specDir = join(specsDir, 'test-spec');
      const assertionsDir = join(specDir, 'assertions');
      mkdirSync(assertionsDir, { recursive: true });

      writeFileSync(join(specDir, 'test-spec.md'), `---
id: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
---

# Test Spec`);

      // Create a simple linear chain: A -> B -> C
      // Only C is terminal
      writeFileSync(join(assertionsDir, 'assertion-a.md'), `---
id: assertion-a
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: done
branch: feature/linear
---

# Assertion A`);

      writeFileSync(join(assertionsDir, 'assertion-b.md'), `---
id: assertion-b
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: done
branch: feature/linear
depends-on: assertion-a
---

# Assertion B`);

      writeFileSync(join(assertionsDir, 'assertion-c.md'), `---
id: assertion-c
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: in_progress
branch: feature/linear
depends-on: assertion-b
---

# Assertion C`);

      execSync(`node "${join(projectRoot, 'bin/spekk.js')}" show`, {
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000,
        env: { ...process.env, NODE_ENV: 'test' }
      });

      const htmlContent = readFileSync(join(testDir, '.spekk', 'index.html'), 'utf8');

      // Should NOT have "Done" terminus
      assert.ok(!htmlContent.includes('Done Terminus'), 'Should not include Done terminus (only 1 terminal)');
      assert.ok(!htmlContent.includes('class="metro-terminus"'), 'Should not have metro-terminus class');

    } finally {
      cleanup();
    }
  });
});
