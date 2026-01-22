import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'fs';
import path from 'path';
import { parseAllSpecs } from '../index.js';

describe('Ignore Markdown Files Without Frontmatter', () => {
  let tempDir;
  
  function setupTestDirectory() {
    tempDir = path.join(process.cwd(), 'temp-ignore-readme-test');
    fs.mkdirSync(tempDir, { recursive: true });
    
    const originalCwd = process.cwd();
    process.chdir(tempDir);
    
    return originalCwd;
  }
  
  function teardownTestDirectory(originalCwd) {
    process.chdir(originalCwd);
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true });
    }
  }

  test('parser ignores README.md files without frontmatter', () => {
    const originalCwd = setupTestDirectory();
    
    try {
      // Create specs directory with proper structure
      fs.mkdirSync('specs/test-spec', { recursive: true });
      fs.mkdirSync('specs/test-spec/assertions');
      
      // Create a valid spec file
      const validSpecContent = `---
id: test-spec
created: 2026-01-22T22:45:00Z
priority: 1
---

# Test Spec

This is a valid spec file.`;

      fs.writeFileSync('specs/test-spec/test-spec.md', validSpecContent);
      
      // Create a valid assertion file
      const validAssertionContent = `---
id: test-assertion
parent: test-spec
created: 2026-01-22T22:46:00Z
priority: 1
status: not_started
---

# Test Assertion

This is a valid assertion file.`;

      fs.writeFileSync('specs/test-spec/assertions/test-assertion.md', validAssertionContent);
      
      // Create README.md without frontmatter (should be ignored)
      const readmeContent = `# README

This is a README file without frontmatter.
It should be completely ignored by the parser.

## Installation

npm install

## Usage

npm run test`;

      fs.writeFileSync('specs/test-spec/README.md', readmeContent);
      
      // Create another docs file without frontmatter in assertions directory
      const docsContent = `# Documentation

This documentation file has no frontmatter.
Should be silently ignored.

## API Reference

function example() {
  return 'test';
}`;

      fs.writeFileSync('specs/test-spec/assertions/docs.md', docsContent);
      
      // Parser should succeed without throwing errors
      const result = parseAllSpecs();
      
      // Should have found 1 spec and 1 assertion (ignoring README and docs)
      assert.equal(result.specs.length, 1);
      assert.equal(result.assertions.length, 1);
      
      // Verify the valid files were parsed correctly
      assert.equal(result.specs[0].id, 'test-spec');
      assert.equal(result.assertions[0].id, 'test-assertion');
      
    } finally {
      teardownTestDirectory(originalCwd);
    }
  });

  test('parser ignores any .md file not starting with frontmatter', () => {
    const originalCwd = setupTestDirectory();
    
    try {
      // Create specs directory
      fs.mkdirSync('specs/example-spec', { recursive: true });
      fs.mkdirSync('specs/example-spec/assertions');
      
      // Create valid spec
      const validSpecContent = `---
id: example-spec
created: 2026-01-22T22:45:00Z
priority: 2
---

# Example Spec`;

      fs.writeFileSync('specs/example-spec/example-spec.md', validSpecContent);
      
      // Create multiple invalid files (no frontmatter)
      const files = [
        'notes.md',
        'CHANGELOG.md', 
        'architecture.md',
        'random-docs.md'
      ];
      
      files.forEach(filename => {
        const content = `# ${filename.replace('.md', '')}

This file has no frontmatter and should be ignored.

Some content here...`;
        
        fs.writeFileSync(`specs/example-spec/${filename}`, content);
      });
      
      // Create invalid assertion files too
      const assertionFiles = [
        'help.md',
        'guide.md'
      ];
      
      assertionFiles.forEach(filename => {
        const content = `# Helper File

No frontmatter here either.

This should also be ignored.`;
        
        fs.writeFileSync(`specs/example-spec/assertions/${filename}`, content);
      });
      
      // Parser should succeed and only find the valid spec
      const result = parseAllSpecs();
      
      assert.equal(result.specs.length, 1);
      assert.equal(result.assertions.length, 0); // No valid assertions
      assert.equal(result.specs[0].id, 'example-spec');
      
    } finally {
      teardownTestDirectory(originalCwd);
    }
  });

  test('parser handles mixed valid and invalid files correctly', () => {
    const originalCwd = setupTestDirectory();
    
    try {
      // Create specs directory
      fs.mkdirSync('specs/mixed-spec', { recursive: true });
      fs.mkdirSync('specs/mixed-spec/assertions');
      
      // Valid spec file
      const validSpecContent = `---
id: mixed-spec
created: 2026-01-22T22:45:00Z
priority: 1
---

# Mixed Spec`;

      fs.writeFileSync('specs/mixed-spec/mixed-spec.md', validSpecContent);
      
      // Valid assertion
      const validAssertionContent = `---
id: valid-assertion
parent: mixed-spec
created: 2026-01-22T22:46:00Z
priority: 1
status: done
---

# Valid Assertion`;

      fs.writeFileSync('specs/mixed-spec/assertions/valid-assertion.md', validAssertionContent);
      
      // Invalid files mixed in
      fs.writeFileSync('specs/mixed-spec/README.md', '# README\n\nNo frontmatter here.');
      fs.writeFileSync('specs/mixed-spec/assertions/README.md', '# Assertion README\n\nAlso no frontmatter.');
      
      // Another valid assertion
      const anotherValidAssertion = `---
id: another-assertion  
parent: mixed-spec
created: 2026-01-22T22:47:00Z
priority: 2
status: in_progress
---

# Another Valid Assertion`;

      fs.writeFileSync('specs/mixed-spec/assertions/another-assertion.md', anotherValidAssertion);
      
      const result = parseAllSpecs();
      
      // Should find 1 spec and 2 valid assertions (ignoring READMEs)
      assert.equal(result.specs.length, 1);
      assert.equal(result.assertions.length, 2);
      
      const assertionIds = result.assertions.map(a => a.id).sort();
      assert.deepEqual(assertionIds, ['another-assertion', 'valid-assertion']);
      
    } finally {
      teardownTestDirectory(originalCwd);
    }
  });

  test('parser still validates frontmatter for files that have it', () => {
    const originalCwd = setupTestDirectory();
    
    try {
      fs.mkdirSync('specs/validation-spec', { recursive: true });
      fs.mkdirSync('specs/validation-spec/assertions');
      
      // Valid spec
      const validSpecContent = `---
id: validation-spec
created: 2026-01-22T22:45:00Z
priority: 1
---

# Validation Spec`;

      fs.writeFileSync('specs/validation-spec/validation-spec.md', validSpecContent);
      
      // Invalid assertion (has frontmatter but missing required field)
      const invalidAssertionContent = `---
id: invalid-assertion
created: 2026-01-22T22:46:00Z
priority: 1
status: not_started
---

# Invalid Assertion - Missing Parent Field`;

      fs.writeFileSync('specs/validation-spec/assertions/invalid-assertion.md', invalidAssertionContent);
      
      // Should still throw error for invalid frontmatter
      assert.throws(() => {
        parseAllSpecs();
      }, {
        message: /Missing required field 'parent'/
      });
      
    } finally {
      teardownTestDirectory(originalCwd);
    }
  });

  test('parser works in directory with only README files', () => {
    const originalCwd = setupTestDirectory();
    
    try {
      fs.mkdirSync('specs', { recursive: true });
      
      // Only create README files (no valid specs)
      fs.writeFileSync('specs/README.md', '# Main README\n\nNo specs here.');
      
      fs.mkdirSync('specs/empty-dir');
      fs.writeFileSync('specs/empty-dir/README.md', '# Empty Dir README');
      fs.writeFileSync('specs/empty-dir/docs.md', '# Documentation');
      
      // Should return empty results without crashing
      const result = parseAllSpecs();
      
      assert.equal(result.specs.length, 0);
      assert.equal(result.assertions.length, 0);
      
    } finally {
      teardownTestDirectory(originalCwd);
    }
  });
});