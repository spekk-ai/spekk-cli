import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import { existsSync, rmSync, mkdirSync, writeFileSync, readFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { tmpdir } from 'node:os';

describe('Searchbar Filters Spec Tree', () => {

  const testDir = join(tmpdir(), `spekk-search-test-${Date.now()}`);
  const __dirname = dirname(fileURLToPath(import.meta.url));
  const projectRoot = join(__dirname, '../..');

  function cleanup() {
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
  }

  function setupProject(specFiles) {
    cleanup();
    mkdirSync(testDir, { recursive: true });

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

    for (const [filename, content] of Object.entries(specFiles)) {
      writeFileSync(join(assertionsDir, filename), content);
    }

    execSync(`node "${join(projectRoot, 'bin/spekk.js')}" show`, {
      encoding: 'utf8',
      cwd: testDir,
      timeout: 5000,
      env: { ...process.env, NODE_ENV: 'test' }
    });

    return readFileSync(join(testDir, '.spekk', 'index.html'), 'utf8');
  }

  test('includes search input with correct placement and placeholder', () => {
    try {
      const html = setupProject({
        'test-assertion.md': `---
id: test-assertion
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Test Assertion`
      });

      // Search input exists
      assert.ok(html.includes('id="spec-search"'), 'Should include search input with id spec-search');
      assert.ok(html.includes('placeholder="Search specs..."'), 'Should have correct placeholder text');
      assert.ok(html.includes('class="search-container"'), 'Should have search container');

      // Search input appears after toggle and before spec tree
      const toggleIndex = html.indexOf('toggle-completed-specs');
      const searchIndex = html.indexOf('id="spec-search"');
      const specTreeIndex = html.indexOf('class="spec-tree"');
      assert.ok(toggleIndex < searchIndex, 'Search input should appear after the toggle');
      assert.ok(searchIndex < specTreeIndex, 'Search input should appear before the spec tree');

    } finally {
      cleanup();
    }
  });

  test('includes search CSS and JavaScript for filtering behavior', () => {
    try {
      const html = setupProject({
        'alpha-assertion.md': `---
id: alpha-assertion
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: done
---

# Alpha Assertion`,
        'beta-assertion.md': `---
id: beta-assertion
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 2
status: not_started
---

# Beta Assertion`
      });

      // CSS for search-hidden class
      assert.ok(html.includes('.spec-item.search-hidden'), 'Should include CSS for hiding spec items');
      assert.ok(html.includes('.assertion-item.search-hidden'), 'Should include CSS for hiding assertion items');

      // JavaScript search initialization
      assert.ok(html.includes('initializeSearch'), 'Should include initializeSearch function');

      // Data attributes for searchable text on spec items and assertion items
      assert.ok(html.includes('data-search-text='), 'Should include data-search-text attributes');

      // Verify searchable text contains assertion identifiers
      assert.ok(html.includes('alpha-assertion'), 'Should include alpha assertion id in searchable text');
      assert.ok(html.includes('beta-assertion'), 'Should include beta assertion id in searchable text');

      // Search does NOT persist in localStorage
      assert.ok(!html.includes('spekkSearch'), 'Search state should not be saved to localStorage');

    } finally {
      cleanup();
    }
  });

  test('search clear collapses auto-expanded specs but not manually expanded ones', () => {
    try {
      const html = setupProject({
        'test-assertion.md': `---
id: test-assertion
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Test Assertion`
      });

      // searchExpandedSpecs Set is declared to track auto-expanded specs
      assert.ok(
        html.includes('searchExpandedSpecs = new Set()'),
        'Should declare searchExpandedSpecs Set to track auto-expanded specs'
      );

      // Before expanding, the spec is added to the tracking Set
      assert.ok(
        html.includes('searchExpandedSpecs.add(specItem)'),
        'Should add spec to searchExpandedSpecs before expanding it'
      );

      // On search clear, auto-expanded specs are collapsed (expanded class removed)
      assert.ok(
        html.includes('searchExpandedSpecs.forEach'),
        'Should iterate searchExpandedSpecs on clear to collapse them'
      );

      // The Set is cleared after collapsing
      assert.ok(
        html.includes('searchExpandedSpecs.clear()'),
        'Should clear the searchExpandedSpecs Set after collapsing'
      );

      // Collapse logic removes expanded class and resets toggle text to right-pointing triangle
      assert.ok(
        html.includes("toggle.textContent = '\\u25B6'") || html.includes("toggle.textContent = '\u25B6'"),
        'Should reset toggle text to collapsed arrow on search clear'
      );

    } finally {
      cleanup();
    }
  });

  test('search-match class overrides completed spec hiding', () => {
    try {
      const html = setupProject({
        'done-assertion.md': `---
id: done-assertion
parent: test-spec
created: 2026-01-22T21:00:00Z
priority: 1
status: done
---

# Done Assertion`
      });

      // CSS rule: .spec-item.search-match must override display with !important
      assert.ok(
        html.includes('.spec-item.search-match'),
        'Should include CSS rule for .spec-item.search-match'
      );
      // The rule must use display: block !important to override .spec-item.completed { display: none }
      const searchMatchRuleStart = html.indexOf('.spec-item.search-match');
      const searchMatchRuleEnd = html.indexOf('}', searchMatchRuleStart);
      const searchMatchRule = html.slice(searchMatchRuleStart, searchMatchRuleEnd + 1);
      assert.ok(
        searchMatchRule.includes('display: block !important') || searchMatchRule.includes('display:block !important'),
        'search-match rule must use display: block !important to override completed hiding'
      );

      // JS adds search-match class when a spec matches during active search
      assert.ok(
        html.includes("classList.add('search-match')"),
        'JS should add search-match class to matching spec items'
      );

      // JS removes search-match class when search is cleared
      assert.ok(
        html.includes("classList.remove('search-match')"),
        'JS should remove search-match class when search is cleared'
      );

    } finally {
      cleanup();
    }
  });
});
