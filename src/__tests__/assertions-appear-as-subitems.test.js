import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import { existsSync, rmSync, mkdirSync, writeFileSync, readFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { tmpdir } from 'node:os';

describe('Assertions Appear as Sub-items', () => {

  const testDir = join(tmpdir(), `spekk-test-assertions-${Date.now()}`);
  const __dirname = dirname(fileURLToPath(import.meta.url));
  const projectRoot = join(__dirname, '../..');

  test('assertions are organized under correct parent spec in generated HTML', () => {
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
    mkdirSync(testDir, { recursive: true });

    try {
      const specsDir = join(testDir, 'specs');

      // Create spec A with assertions 1,2
      const specADir = join(specsDir, 'spec-a');
      const assertionsADir = join(specADir, 'assertions');
      mkdirSync(assertionsADir, { recursive: true });

      writeFileSync(join(specADir, 'spec-a.md'), `---
id: spec-a
created: 2026-01-22T21:00:00Z
priority: 1
---

# Spec A`);

      writeFileSync(join(assertionsADir, 'assertion-a1.md'), `---
id: assertion-a1
parent: spec-a
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Assertion A1`);

      writeFileSync(join(assertionsADir, 'assertion-a2.md'), `---
id: assertion-a2
parent: spec-a
created: 2026-01-22T21:01:00Z
priority: 2
status: not_started
---

# Assertion A2`);

      // Create spec B with assertion 3
      const specBDir = join(specsDir, 'spec-b');
      const assertionsBDir = join(specBDir, 'assertions');
      mkdirSync(assertionsBDir, { recursive: true });

      writeFileSync(join(specBDir, 'spec-b.md'), `---
id: spec-b
created: 2026-01-22T22:00:00Z
priority: 2
---

# Spec B`);

      writeFileSync(join(assertionsBDir, 'assertion-b1.md'), `---
id: assertion-b1
parent: spec-b
created: 2026-01-22T22:00:00Z
priority: 1
status: not_started
---

# Assertion B1`);

      // Run spekk show command
      execSync(`node "${join(projectRoot, 'bin/spekk.js')}" show`, {
        encoding: 'utf8',
        cwd: testDir,
        timeout: 5000
      });

      const htmlFile = join(testDir, '.spekk', 'index.html');
      const htmlContent = readFileSync(htmlFile, 'utf8');

      // Extract assertions for spec A
      const specAMatch = htmlContent.match(/id="assertions-spec-a"[^>]*>(.*?)(?=<\/ul>)/s);
      assert.ok(specAMatch, 'Should find assertions container for spec-a');
      const specAAssertions = specAMatch[1];

      // Extract assertions for spec B
      const specBMatch = htmlContent.match(/id="assertions-spec-b"[^>]*>(.*?)(?=<\/ul>)/s);
      assert.ok(specBMatch, 'Should find assertions container for spec-b');
      const specBAssertions = specBMatch[1];

      // Verify correct grouping
      assert.ok(specAAssertions.includes('assertion-a1'), 'Spec A should contain assertion-a1');
      assert.ok(specAAssertions.includes('assertion-a2'), 'Spec A should contain assertion-a2');
      assert.ok(!specAAssertions.includes('assertion-b1'), 'Spec A should NOT contain assertion-b1');

      assert.ok(specBAssertions.includes('assertion-b1'), 'Spec B should contain assertion-b1');
      assert.ok(!specBAssertions.includes('assertion-a1'), 'Spec B should NOT contain assertion-a1');

    } finally {
      if (existsSync(testDir)) {
        rmSync(testDir, { recursive: true, force: true });
      }
    }
  });
});
