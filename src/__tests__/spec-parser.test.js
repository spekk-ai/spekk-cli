import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import fs from 'fs';
import path from 'path';
import { parseFrontmatter } from '../parser/index.js';

describe('Spec Parser', () => {
  test('parser script exists and is executable', () => {
    const cliPath = path.join(process.cwd(), 'src', 'parser', 'cli.js');
    assert.ok(fs.existsSync(cliPath), 'src/parser/cli.js should exist');
    
    // Check if it's executable by running it via npm
    const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
    const parsed = JSON.parse(result);
    
    // Should return valid JSON structure
    assert.ok(typeof parsed === 'object', 'Should return JSON object');
  });

  test('outputs valid JSON format', () => {
    const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
    
    // Should be parseable JSON
    const parsed = JSON.parse(result);
    assert.ok(typeof parsed === 'object', 'Output should be valid JSON object');
    
    // Should have expected structure for success case
    if (parsed.type === 'assertion') {
      assert.ok(parsed.id, 'Should have id field');
      assert.ok(parsed.parent, 'Should have parent field');
      assert.ok(parsed.file, 'Should have file field');
      assert.ok(typeof parsed.priority === 'number', 'Priority should be number');
      assert.ok(parsed.status, 'Should have status field');
      assert.ok(parsed.title, 'Should have title field');
      assert.ok(parsed.content, 'Should have content field');
    }
  });

  test('identifies next priority assertion correctly', () => {
    const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
    const parsed = JSON.parse(result);
    
    if (parsed.type === 'assertion') {
      // Should return highest priority incomplete assertion
      assert.ok([1, 2, 3].includes(parsed.priority), 'Priority should be 1, 2, or 3');
      assert.ok(['not_started', 'in_progress'].includes(parsed.status), 'Should return incomplete assertion');
    }
  });

  test('validates status values - accepts all valid values', () => {
    // Test that parser accepts valid status values
    const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
    const parsed = JSON.parse(result);
    
    if (parsed.type === 'assertion') {
      assert.ok(['not_started', 'in_progress', 'done'].includes(parsed.status), 
        'Status should be valid value');
    }
  });

  test('validates timestamp format', () => {
    const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
    const parsed = JSON.parse(result);
    
    if (parsed.type === 'assertion') {
      // Check if we can parse the assertion file to get timestamp
      const content = fs.readFileSync(parsed.file, 'utf8');
      
      // Should have created timestamp in ISO format
      const timestampMatch = content.match(/created:\s*(.+)/);
      if (timestampMatch) {
        const timestamp = timestampMatch[1].trim();
        // Should match ISO 8601 format
        assert.ok(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/.test(timestamp), 
          'Created timestamp should be ISO 8601 format');
      }
    }
  });

  test('enforces folder structure - no flat spec files', () => {
    // Check that there are no flat .md files at specs/*.md level
    const specsDir = path.join(process.cwd(), 'specs');
    const files = fs.readdirSync(specsDir);
    
    const flatMdFiles = files.filter(file => file.endsWith('.md'));
    assert.strictEqual(flatMdFiles.length, 0, 
      `Found flat .md files in specs/: ${flatMdFiles.join(', ')}. All specs must be in folders.`);
  });

  test('enforces folder structure - valid spec directories', () => {
    // Check that specs follow expected structure
    const specsDir = path.join(process.cwd(), 'specs');
    const specDirs = fs.readdirSync(specsDir).filter(dir => {
      const dirPath = path.join(specsDir, dir);
      return fs.statSync(dirPath).isDirectory();
    });
    
    for (const specDir of specDirs) {
      const specFile = path.join(specsDir, specDir, `${specDir}.md`);
      const assertionsDir = path.join(specsDir, specDir, 'assertions');
      
      // Should have spec file with matching name
      assert.ok(fs.existsSync(specFile), 
        `Spec file ${specDir}/${specDir}.md should exist`);
      
      // Should have assertions directory
      assert.ok(fs.existsSync(assertionsDir), 
        `Assertions directory ${specDir}/assertions/ should exist`);
    }
  });

  describe('YAML Frontmatter Parsing', () => {
    test('parses well-formed YAML frontmatter correctly', () => {
      const content = `---
id: test-spec
created: 2026-01-20T16:00:00Z
priority: 1
status: not_started
---

# Test Spec Title

This is markdown content.`;
      
      const result = parseFrontmatter(content);
      
      assert.strictEqual(result.data.id, 'test-spec');
      assert.strictEqual(result.data.created, '2026-01-20T16:00:00Z');
      assert.strictEqual(result.data.priority, 1);
      assert.strictEqual(result.data.status, 'not_started');
      assert.ok(result.content.includes('# Test Spec Title'));
      assert.ok(result.content.includes('This is markdown content.'));
    });

    test('separates frontmatter from markdown content correctly', () => {
      const content = `---
id: separation-test
priority: 2
---

# Markdown Title

Line 1 of content
Line 2 of content`;
      
      const result = parseFrontmatter(content);
      
      // Frontmatter should be parsed
      assert.strictEqual(result.data.id, 'separation-test');
      assert.strictEqual(result.data.priority, 2);
      
      // Content should only contain markdown (no YAML delimiters)
      assert.ok(!result.content.includes('---'));
      assert.ok(!result.content.includes('id: separation-test'));
      assert.ok(result.content.includes('# Markdown Title'));
      assert.ok(result.content.includes('Line 1 of content'));
    });

    test('handles different YAML value types', () => {
      const content = `---
string_field: example-value
number_field: 42
boolean_true: true
boolean_false: false
quoted_string: "quoted value"
---

Content here.`;
      
      const result = parseFrontmatter(content);
      
      assert.strictEqual(result.data.string_field, 'example-value');
      assert.strictEqual(result.data.number_field, 42);
      assert.strictEqual(result.data.boolean_true, true);
      assert.strictEqual(result.data.boolean_false, false);
      assert.strictEqual(result.data.quoted_string, 'quoted value');
    });

    test('throws error for missing opening frontmatter delimiter', () => {
      const content = `id: test-spec
priority: 1
---

# Content`;
      
      assert.throws(() => {
        parseFrontmatter(content);
      }, /File must start with --- YAML frontmatter delimiter/);
    });

    test('throws error for missing closing frontmatter delimiter', () => {
      const content = `---
id: test-spec
priority: 1

# Content without closing delimiter`;
      
      assert.throws(() => {
        parseFrontmatter(content);
      }, /Missing closing --- delimiter for YAML frontmatter/);
    });

    test('handles empty frontmatter correctly', () => {
      const content = `---
---

# Just Content

This has empty frontmatter.`;
      
      const result = parseFrontmatter(content);
      
      // Should have empty data object
      assert.deepStrictEqual(result.data, {});
      assert.ok(result.content.includes('# Just Content'));
    });

    test('handles multi-line markdown content', () => {
      const content = `---
id: multiline-test
priority: 1
---

# Title

Paragraph 1 with multiple words.

## Subtitle

- List item 1
- List item 2

More content here.`;
      
      const result = parseFrontmatter(content);
      
      assert.strictEqual(result.data.id, 'multiline-test');
      assert.ok(result.content.includes('Paragraph 1 with multiple words.'));
      assert.ok(result.content.includes('## Subtitle'));
      assert.ok(result.content.includes('- List item 1'));
    });

    test('works with real spec and assertion files', () => {
      // Test with actual files from the codebase
      const specsDir = path.join(process.cwd(), 'specs');
      const specDirs = fs.readdirSync(specsDir).filter(dir => {
        const dirPath = path.join(specsDir, dir);
        return fs.statSync(dirPath).isDirectory();
      });

      let testedFiles = 0;
      
      for (const specDir of specDirs) {
        // Test spec file
        const specFile = path.join(specsDir, specDir, `${specDir}.md`);
        if (fs.existsSync(specFile)) {
          const content = fs.readFileSync(specFile, 'utf8');
          const result = parseFrontmatter(content);
          
          assert.ok(typeof result.data === 'object', `Spec file ${specFile} should have valid frontmatter`);
          assert.ok(typeof result.content === 'string', `Spec file ${specFile} should have content`);
          testedFiles++;
        }
        
        // Test assertion files
        const assertionsDir = path.join(specsDir, specDir, 'assertions');
        if (fs.existsSync(assertionsDir)) {
          const assertionFiles = fs.readdirSync(assertionsDir).filter(file => file.endsWith('.md'));
          
          for (const assertionFile of assertionFiles) {
            const assertionPath = path.join(assertionsDir, assertionFile);
            const content = fs.readFileSync(assertionPath, 'utf8');
            const result = parseFrontmatter(content);
            
            assert.ok(typeof result.data === 'object', `Assertion file ${assertionPath} should have valid frontmatter`);
            assert.ok(typeof result.content === 'string', `Assertion file ${assertionPath} should have content`);
            testedFiles++;
          }
        }
      }
      
      assert.ok(testedFiles > 0, 'Should have tested at least one file');
    });
  });
});