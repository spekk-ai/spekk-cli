import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'fs';
import path from 'path';
import { parseFrontmatter } from '../index.js';

describe('YAML Frontmatter Parsing', () => {
  test('parses well-formed YAML frontmatter correctly', () => {
    const content = `---
id: test-spec
parent: example-parent
created: 2026-01-20T16:00:00Z
priority: 1
status: not_started
---

# Test Content

This is the markdown content.`;

    const result = parseFrontmatter(content);
    
    assert.equal(result.data.id, 'test-spec');
    assert.equal(result.data.parent, 'example-parent');
    assert.equal(result.data.created, '2026-01-20T16:00:00Z');
    assert.equal(result.data.priority, 1);
    assert.equal(result.data.status, 'not_started');
  });

  test('separates frontmatter from markdown content correctly', () => {
    const content = `---
id: test-spec
priority: 1
---

# Test Heading

This is the markdown content with multiple lines.
It should be preserved exactly.`;

    const result = parseFrontmatter(content);
    
    const expectedContent = `
# Test Heading

This is the markdown content with multiple lines.
It should be preserved exactly.`;
    
    assert.equal(result.content, expectedContent);
    assert.equal(result.data.id, 'test-spec');
    assert.equal(result.data.priority, 1);
  });

  test('handles different YAML value types', () => {
    const content = `---
string_value: simple-string
quoted_string: "quoted value"
number_value: 42
boolean_true: true
boolean_false: false
priority: 3
---

# Test Content`;

    const result = parseFrontmatter(content);
    
    assert.equal(result.data.string_value, 'simple-string');
    assert.equal(result.data.quoted_string, 'quoted value');
    assert.equal(result.data.number_value, 42);
    assert.equal(result.data.boolean_true, true);
    assert.equal(result.data.boolean_false, false);
    assert.equal(result.data.priority, 3);
    
    // Verify types
    assert.equal(typeof result.data.string_value, 'string');
    assert.equal(typeof result.data.quoted_string, 'string');
    assert.equal(typeof result.data.number_value, 'number');
    assert.equal(typeof result.data.boolean_true, 'boolean');
    assert.equal(typeof result.data.boolean_false, 'boolean');
  });

  test('throws error for missing opening frontmatter delimiter', () => {
    const content = `id: test-spec
priority: 1
---

# Test Content`;

    assert.throws(() => {
      parseFrontmatter(content);
    }, {
      message: 'File must start with --- YAML frontmatter delimiter'
    });
  });

  test('throws error for missing closing frontmatter delimiter', () => {
    const content = `---
id: test-spec
priority: 1

# Test Content

This has no closing delimiter.`;

    assert.throws(() => {
      parseFrontmatter(content);
    }, {
      message: 'Missing closing --- delimiter for YAML frontmatter'
    });
  });

  test('handles empty frontmatter correctly', () => {
    const content = `---
---

# Test Content

This has empty frontmatter.`;

    const result = parseFrontmatter(content);
    
    assert.deepEqual(result.data, {});
    assert.equal(result.content, `
# Test Content

This has empty frontmatter.`);
  });

  test('handles multi-line markdown content', () => {
    const content = `---
id: multi-line-test
priority: 2
---

# Main Heading

## Sub Heading

This is a paragraph with multiple lines.
It continues here.

- List item 1
- List item 2

\`\`\`javascript
const code = 'example';
\`\`\`

Final paragraph.`;

    const result = parseFrontmatter(content);
    
    const expectedContent = `
# Main Heading

## Sub Heading

This is a paragraph with multiple lines.
It continues here.

- List item 1
- List item 2

\`\`\`javascript
const code = 'example';
\`\`\`

Final paragraph.`;
    
    assert.equal(result.content, expectedContent);
    assert.equal(result.data.id, 'multi-line-test');
    assert.equal(result.data.priority, 2);
  });

  test('works with real spec and assertion files', () => {
    // Test with actual spec file structure
    const tempDir = path.join(process.cwd(), 'temp-real-file-test');
    
    try {
      fs.mkdirSync(tempDir, { recursive: true });
      
      const realSpecContent = `---
id: real-spec-example
created: 2026-01-21T10:30:00Z
priority: 1
---

# Real Spec Example

## Overview

This is a real specification file that tests the frontmatter parser
with realistic content structure.

## Requirements

- Must parse frontmatter correctly
- Must preserve markdown formatting
- Must handle various content types

## Implementation Notes

Code blocks should be preserved:

\`\`\`bash
npm test
\`\`\`

Lists should work:
1. First item
2. Second item
3. Third item`;

      const realAssertionContent = `---
id: real-assertion-example
parent: real-spec-example
created: 2026-01-21T10:31:00Z
priority: 2
status: in_progress
---

# Real Assertion Example

**What Must Be True**

The parser must handle this real assertion file correctly.

**Success Criteria**
- ✅ Frontmatter parsed
- ✅ Content preserved
- ⏳ Status tracked

**Tests:** src/parser/__tests__/frontmatter-parsing.test.js`;

      const specFile = path.join(tempDir, 'real-spec.md');
      const assertionFile = path.join(tempDir, 'real-assertion.md');
      
      fs.writeFileSync(specFile, realSpecContent);
      fs.writeFileSync(assertionFile, realAssertionContent);
      
      // Test spec file parsing
      const specContent = fs.readFileSync(specFile, 'utf8');
      const specResult = parseFrontmatter(specContent);
      
      assert.equal(specResult.data.id, 'real-spec-example');
      assert.equal(specResult.data.priority, 1);
      assert.ok(specResult.content.includes('# Real Spec Example'));
      assert.ok(specResult.content.includes('```bash'));
      assert.ok(specResult.content.includes('npm test'));
      
      // Test assertion file parsing
      const assertionContentRaw = fs.readFileSync(assertionFile, 'utf8');
      const assertionResult = parseFrontmatter(assertionContentRaw);
      
      assert.equal(assertionResult.data.id, 'real-assertion-example');
      assert.equal(assertionResult.data.parent, 'real-spec-example');
      assert.equal(assertionResult.data.status, 'in_progress');
      assert.equal(assertionResult.data.priority, 2);
      assert.ok(assertionResult.content.includes('# Real Assertion Example'));
      assert.ok(assertionResult.content.includes('**What Must Be True**'));
      assert.ok(assertionResult.content.includes('✅ Frontmatter parsed'));
      
    } finally {
      if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
    }
  });

  test('handles frontmatter with colons in values', () => {
    const content = `---
id: colon-test
url: https://example.com:8080/path
timestamp: 2026-01-21T10:30:15Z
description: "Value with: colons in it"
---

# Test Content`;

    const result = parseFrontmatter(content);
    
    assert.equal(result.data.id, 'colon-test');
    assert.equal(result.data.url, 'https://example.com:8080/path');
    assert.equal(result.data.timestamp, '2026-01-21T10:30:15Z');
    assert.equal(result.data.description, 'Value with: colons in it');
  });

  test('handles whitespace variations in frontmatter', () => {
    const content = `---
id:test-whitespace
  spaced_key  :  spaced_value  
priority  : 1
status:   not_started   
---

# Test Content`;

    const result = parseFrontmatter(content);
    
    assert.equal(result.data.id, 'test-whitespace');
    assert.equal(result.data.spaced_key, 'spaced_value');
    assert.equal(result.data.priority, 1);
    assert.equal(result.data.status, 'not_started');
  });

  test('preserves empty lines and formatting in markdown content', () => {
    const content = `---
id: formatting-test
priority: 1
---

# Heading With Empty Lines


This paragraph has empty lines above and below.


## Another Section

Content here.

`;

    const result = parseFrontmatter(content);
    
    const expectedContent = `
# Heading With Empty Lines


This paragraph has empty lines above and below.


## Another Section

Content here.

`;
    
    assert.equal(result.content, expectedContent);
  });

  test('handles numbers as strings when not purely numeric', () => {
    const content = `---
id: mixed-numbers
version: 1.0.1
port: 8080
mixed: 42abc
pure_number: 123
---

# Test Content`;

    const result = parseFrontmatter(content);
    
    assert.equal(result.data.version, '1.0.1'); // string because of dots
    assert.equal(result.data.port, 8080); // number
    assert.equal(result.data.mixed, '42abc'); // string because of letters
    assert.equal(result.data.pure_number, 123); // number
    
    assert.equal(typeof result.data.version, 'string');
    assert.equal(typeof result.data.port, 'number');
    assert.equal(typeof result.data.mixed, 'string');
    assert.equal(typeof result.data.pure_number, 'number');
  });
});