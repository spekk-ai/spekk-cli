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
});
