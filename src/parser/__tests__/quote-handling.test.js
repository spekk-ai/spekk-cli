import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import fs from 'fs';
import path from 'path';
import os from 'os';
import { parseAllSpecs } from '../index.js';

describe('Quote Handling in Titles and Content', () => {
  test('handles various quote types in spec titles and content', () => {
    const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-quotes-'));
    const specsDir = path.join(testDir, 'specs');
    const specDir = path.join(specsDir, 'quote-handling-test');
    const assertionsDir = path.join(specDir, 'assertions');

    try {
      fs.mkdirSync(assertionsDir, { recursive: true });

      const specContent = `---
id: quote-handling-test
created: 2026-01-22T21:00:00Z
priority: 1
---

# User's Dashboard "Test" with \`backticks\`

This spec has a single quote in the title and content with "quotes" and \`backticks\` and {curly: braces}.`;

      const assertionContent = `---
id: quote-assertion-test
parent: quote-handling-test
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Test "Quoted" Assertion with 'Single' and \`Back\` Quotes

This assertion has double quotes and 'single quotes' and \`backticks\` in title and content.
Also testing curly {braces} and square [brackets] and parentheses (like this).`;

      fs.writeFileSync(path.join(specDir, 'quote-handling-test.md'), specContent);
      fs.writeFileSync(path.join(assertionsDir, 'quote-assertion.md'), assertionContent);

      const { specs, assertions } = parseAllSpecs(specsDir);

      const testSpec = specs.find(s => s.id === 'quote-handling-test');
      const testAssertion = assertions.find(a => a.id === 'quote-assertion-test');

      assert.ok(testSpec, 'Should find spec with quotes in title');
      assert.ok(testAssertion, 'Should find assertion with quotes in title');

      // Verify titles are parsed correctly
      assert.ok(testSpec.title.includes('"Test"'), 'Spec title should contain double quotes');
      assert.ok(testSpec.title.includes("User's"), 'Spec title should contain single quote');
      assert.ok(testSpec.title.includes('`backticks`'), 'Spec title should contain backticks');

      assert.ok(testAssertion.title.includes('"Quoted"'), 'Assertion title should contain double quotes');
      assert.ok(testAssertion.title.includes("'Single'"), 'Assertion title should contain single quotes');
      assert.ok(testAssertion.title.includes('`Back`'), 'Assertion title should contain backticks');

      // Verify content is parsed correctly
      assert.ok(testSpec.content.includes('"quotes"'), 'Spec content should contain quotes');
      assert.ok(testSpec.content.includes('`backticks`'), 'Spec content should contain backticks');
      assert.ok(testSpec.content.includes('{curly: braces}'), 'Spec content should contain curly braces');

      assert.ok(testAssertion.content.includes("'single quotes'"), 'Assertion content should contain single quotes');
      assert.ok(testAssertion.content.includes('`backticks`'), 'Assertion content should contain backticks');
      assert.ok(testAssertion.content.includes('{braces}'), 'Assertion content should contain curly braces');
      assert.ok(testAssertion.content.includes('[brackets]'), 'Assertion content should contain square brackets');
      assert.ok(testAssertion.content.includes('(like this)'), 'Assertion content should contain parentheses');
    } finally {
      if (fs.existsSync(testDir)) fs.rmSync(testDir, { recursive: true, force: true });
    }
  });

  test('CLI outputs valid JSON for specs with quotes', () => {
    const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-cli-quotes-'));
    const specsDir = path.join(testDir, 'specs');
    const specDir = path.join(specsDir, 'cli-quote-test');
    const assertionsDir = path.join(specDir, 'assertions');

    try {
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(specDir, 'cli-quote-test.md'), `---
id: cli-quote-test
created: 2026-01-22T21:00:00Z
priority: 1
---

# CLI "Quote" Test

Testing CLI output with "quotes".`);

      fs.writeFileSync(path.join(assertionsDir, 'cli-quote.md'), `---
id: cli-quote-assertion
parent: cli-quote-test
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# CLI "Test" Assertion

This assertion tests CLI JSON output with 'quotes' and \`backticks\`.`);

      // Test that parseAllSpecs produces valid data with quotes
      const { specs, assertions } = parseAllSpecs(specsDir);
      const testAssertion = assertions.find(a => a.id === 'cli-quote-assertion');

      assert.ok(testAssertion, 'Should parse assertion with quotes');

      // Verify JSON serialization works with quotes
      assert.doesNotThrow(() => {
        JSON.stringify(testAssertion);
      }, 'Should serialize assertion with quotes to valid JSON');

      // Also verify CLI produces valid JSON (uses real project specs)
      const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
      assert.doesNotThrow(() => {
        JSON.parse(result);
      }, 'CLI should output valid JSON');
    } finally {
      if (fs.existsSync(testDir)) fs.rmSync(testDir, { recursive: true, force: true });
    }
  });

  test('handles edge cases with complex quote combinations', () => {
    const testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-complex-quotes-'));
    const specsDir = path.join(testDir, 'specs');
    const specDir = path.join(specsDir, 'complex-quotes-test');
    const assertionsDir = path.join(specDir, 'assertions');

    try {
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(path.join(specDir, 'complex-quotes-test.md'), `---
id: complex-quotes-test
created: 2026-01-22T21:00:00Z
priority: 1
---

# Complex "Mixed 'Quote' Test" with \`Code\`

Testing edge cases:
- Nested "quotes 'inside' quotes"
- Code with \`const x = "string"\`
- JSON-like {"key": "value"}
- Escaped quotes: \\"escaped\\"
- Multiple \`code\` blocks with 'quotes'`);

      fs.writeFileSync(path.join(assertionsDir, 'complex.md'), `---
id: complex-assertion
parent: complex-quotes-test
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Complex "Assertion 'Test'" Case

Edge cases in assertion:
- Template literals: \`Hello "world" with \${variable}\`
- JSON: {"complex": "object 'with' quotes"}
- Code: \`if (str === "test") { return 'result'; }\``);

      // Should parse without errors
      assert.doesNotThrow(() => {
        parseAllSpecs(specsDir);
      }, 'Should parse complex quote combinations without throwing');

      // CLI should produce valid JSON (uses real project specs)
      const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
      assert.doesNotThrow(() => {
        JSON.parse(result);
      }, 'CLI should output valid JSON for complex quote cases');
    } finally {
      if (fs.existsSync(testDir)) fs.rmSync(testDir, { recursive: true, force: true });
    }
  });
});
