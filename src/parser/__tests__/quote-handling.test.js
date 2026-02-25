import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import fs from 'fs';
import path from 'path';
import { parseAllSpecs } from '../index.js';

// TODO: Fix test isolation - tests fail due to cross-contamination from shared specs/ directory
// See GitHub issue #29
describe.skip('Quote Handling in Titles and Content', () => {
  test('handles various quote types in spec titles and content', () => {
    const tempDir = path.join(process.cwd(), 'temp-quote-handling-test');
    const assertionsDir = path.join(tempDir, 'assertions');
    
    try {
      fs.mkdirSync(tempDir, { recursive: true });
      fs.mkdirSync(assertionsDir, { recursive: true });
      
      // Spec with quotes in title and content
      const specContent = `---
id: quote-handling-test
created: 2026-01-22T21:00:00Z
priority: 1
---

# User's Dashboard "Test" with \`backticks\`

This spec has a single quote in the title and content with "quotes" and \`backticks\` and {curly: braces}.`;
      
      // Assertion with quotes in title and content
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
      
      fs.writeFileSync(path.join(tempDir, 'quote-handling-test.md'), specContent);
      fs.writeFileSync(path.join(assertionsDir, 'quote-assertion.md'), assertionContent);
      
      const originalSpecsPath = path.join(process.cwd(), 'specs', 'quote-handling-test');
      fs.symlinkSync(tempDir, originalSpecsPath);
      
      try {
        // Test parsing doesn't throw errors
        const { specs, assertions } = parseAllSpecs();
        
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
        fs.unlinkSync(originalSpecsPath);
      }
      
    } finally {
      if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
    }
  });

  test('CLI outputs valid JSON for specs with quotes', () => {
    const tempDir = path.join(process.cwd(), 'temp-cli-quote-test');
    const assertionsDir = path.join(tempDir, 'assertions');
    
    try {
      fs.mkdirSync(tempDir, { recursive: true });
      fs.mkdirSync(assertionsDir, { recursive: true });
      
      const specContent = `---
id: cli-quote-test
created: 2026-01-22T21:00:00Z
priority: 1
---

# CLI "Quote" Test

Testing CLI output with "quotes".`;
      
      const assertionContent = `---
id: cli-quote-assertion
parent: cli-quote-test
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# CLI "Test" Assertion

This assertion tests CLI JSON output with 'quotes' and \`backticks\`.`;
      
      fs.writeFileSync(path.join(tempDir, 'cli-quote-test.md'), specContent);
      fs.writeFileSync(path.join(assertionsDir, 'cli-quote.md'), assertionContent);
      
      const originalSpecsPath = path.join(process.cwd(), 'specs', 'cli-quote-test');
      fs.symlinkSync(tempDir, originalSpecsPath);
      
      try {
        // Test CLI produces valid JSON
        const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
        
        let parsed;
        assert.doesNotThrow(() => {
          parsed = JSON.parse(result);
        }, 'CLI should output valid JSON even with quotes in content');
        
        // If this assertion was selected, verify quotes are properly escaped in JSON
        if (parsed.type === 'assertion' && parsed.id === 'cli-quote-assertion') {
          assert.ok(parsed.title.includes('"Test"'), 'JSON title should contain quotes');
          assert.ok(parsed.content.includes("'quotes'"), 'JSON content should contain quotes');
          assert.ok(parsed.content.includes('`backticks`'), 'JSON content should contain backticks');
          assert.ok(parsed.spec.title.includes('"Quote"'), 'JSON spec title should contain quotes');
        }
        
      } finally {
        fs.unlinkSync(originalSpecsPath);
      }
      
    } finally {
      if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
    }
  });

  test('handles edge cases with complex quote combinations', () => {
    const tempDir = path.join(process.cwd(), 'temp-complex-quotes-test');
    const assertionsDir = path.join(tempDir, 'assertions');
    
    try {
      fs.mkdirSync(tempDir, { recursive: true });
      fs.mkdirSync(assertionsDir, { recursive: true });
      
      // Test complex quote combinations that could break parsing
      const complexContent = `---
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
- Multiple \`code\` blocks with 'quotes'`;

      const complexAssertion = `---
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
- Code: \`if (str === "test") { return 'result'; }\``;
      
      fs.writeFileSync(path.join(tempDir, 'complex-quotes-test.md'), complexContent);
      fs.writeFileSync(path.join(assertionsDir, 'complex.md'), complexAssertion);
      
      const originalSpecsPath = path.join(process.cwd(), 'specs', 'complex-quotes-test');
      fs.symlinkSync(tempDir, originalSpecsPath);
      
      try {
        // Should parse without errors
        assert.doesNotThrow(() => {
          parseAllSpecs();
        }, 'Should parse complex quote combinations without throwing');
        
        // CLI should produce valid JSON
        const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
        assert.doesNotThrow(() => {
          JSON.parse(result);
        }, 'CLI should output valid JSON for complex quote cases');
        
      } finally {
        fs.unlinkSync(originalSpecsPath);
      }
      
    } finally {
      if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
    }
  });
});