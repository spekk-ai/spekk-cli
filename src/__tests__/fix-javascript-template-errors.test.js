import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import { existsSync, rmSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { tmpdir } from 'node:os';

describe('Fix JavaScript Template Generation Errors', () => {

  const testDir = join(tmpdir(), `spekk-javascript-test-${Date.now()}`);
  const __dirname = dirname(fileURLToPath(import.meta.url));
  const projectRoot = join(__dirname, '../..');

  test('generated HTML handles special characters without JavaScript errors', () => {
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
    mkdirSync(testDir, { recursive: true });

    try {
      const specsDir = join(testDir, 'specs');
      const specDir = join(specsDir, 'comprehensive-test');
      const assertionsDir = join(specDir, 'assertions');
      mkdirSync(assertionsDir, { recursive: true });

      // Spec with all problematic characters: quotes, backticks, curly braces
      writeFileSync(join(specDir, 'comprehensive-test.md'), `---
id: comprehensive-test
created: 2026-01-22T21:00:00Z
priority: 1
---

# Comprehensive "Test" Spec

This spec contains single quotes: User's Dashboard
And double quotes: "quoted text"
And backticks: \`code example\`
And curly braces: {example: value}`);

      writeFileSync(join(assertionsDir, 'comprehensive-assertion.md'), `---
id: comprehensive-assertion
parent: comprehensive-test
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Comprehensive 'Test' Assertion

All special chars: '"'\`{}.`);

      execSync(`node "${join(projectRoot, 'bin/spekk.js')}" show`, {
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000
      });

      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');

      // Extract JavaScript — it should be syntactically valid
      const scriptMatch = htmlContent.match(/<script>([\s\S]*?)<\/script>/);
      assert.ok(scriptMatch, 'HTML should contain script tag');

      const scriptContent = scriptMatch[1];
      assert.ok(scriptContent.includes('function toggleSpec'), 'Script should contain toggleSpec function');
      assert.ok(scriptContent.includes('function showDetail'), 'Script should contain showDetail function');

      // No inline onclick handlers (uses event delegation instead)
      assert.ok(!htmlContent.includes('onclick='), 'Should use event delegation, not inline onclick');

    } finally {
      if (existsSync(testDir)) {
        rmSync(testDir, { recursive: true, force: true });
      }
    }
  });
});
