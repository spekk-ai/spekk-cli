import { test } from 'node:test';
import assert from 'node:assert';
import fs from 'fs';
import path from 'path';
import { parseAllSpecs } from '../index.js';

test('Status Consistency Validation', async (t) => {
  
  const tempDir = path.join(process.cwd(), 'test-temp-specs');
  
  // Clean up before starting
  if (fs.existsSync(tempDir)) {
    fs.rmSync(tempDir, { recursive: true });
  }
  fs.mkdirSync(tempDir);
  
  // Change to temp directory for testing
  const originalCwd = process.cwd();
  process.chdir(tempDir);
  
  try {
    await t.test('parser automatically synchronizes parent spec status', () => {
      // Create a specs directory structure
      fs.mkdirSync('specs');
      fs.mkdirSync('specs/test-spec');
      fs.mkdirSync('specs/test-spec/assertions');
      
      // Create parent spec with manual status
      fs.writeFileSync('specs/test-spec/test-spec.md', 
`---
id: test-spec
created: 2026-01-01T00:00:00Z
priority: 1
status: not_started
---

# Test Spec

This is a test spec.
`);
      
      // Create child assertions with mixed statuses
      fs.writeFileSync('specs/test-spec/assertions/assertion-1.md', 
`---
id: assertion-1
parent: test-spec
created: 2026-01-01T00:00:00Z
priority: 1
status: done
---

# Assertion 1

This assertion is done.
`);
      
      fs.writeFileSync('specs/test-spec/assertions/assertion-2.md', 
`---
id: assertion-2
parent: test-spec
created: 2026-01-01T00:00:00Z
priority: 1
status: in_progress
---

# Assertion 2

This assertion is in progress.
`);
      
      // Parse specs
      const { specs, assertions } = parseAllSpecs();
      
      // Verify that parent status was automatically computed as 'in_progress'
      assert.strictEqual(specs.length, 1);
      assert.strictEqual(specs[0].id, 'test-spec');
      assert.strictEqual(specs[0].status, 'in_progress', 
        'Parent status should be automatically computed as in_progress when child assertions are mixed');
    });

    await t.test('parser handles failed child forcing parent to failed', () => {
      // Update one assertion to failed status
      fs.writeFileSync('specs/test-spec/assertions/assertion-2.md', 
`---
id: assertion-2
parent: test-spec
created: 2026-01-01T00:00:00Z
priority: 1
status: failed
---

# Assertion 2

This assertion has failed.
`);
      
      // Parse specs
      const { specs } = parseAllSpecs();
      
      // Verify that parent status is now 'failed'
      assert.strictEqual(specs[0].status, 'failed', 
        'Parent status should be failed when any child is failed');
    });

    await t.test('parser sets parent to done when all children are done', () => {
      // Update failed assertion to done
      fs.writeFileSync('specs/test-spec/assertions/assertion-2.md', 
`---
id: assertion-2
parent: test-spec
created: 2026-01-01T00:00:00Z
priority: 1
status: done
---

# Assertion 2

This assertion is now done.
`);
      
      // Parse specs
      const { specs } = parseAllSpecs();
      
      // Verify that parent status is now 'done'
      assert.strictEqual(specs[0].status, 'done', 
        'Parent status should be done when all children are done');
    });

    await t.test('parser ignores draft children in status computation', () => {
      // Add a draft assertion
      fs.writeFileSync('specs/test-spec/assertions/assertion-3.md', 
`---
id: assertion-3
parent: test-spec
created: 2026-01-01T00:00:00Z
priority: 1
status: draft
---

# Assertion 3

This assertion is a draft.
`);
      
      // Parse specs
      const { specs } = parseAllSpecs();
      
      // Verify that parent status is still 'done' (draft ignored)
      assert.strictEqual(specs[0].status, 'done', 
        'Parent status should ignore draft children');
    });

    await t.test('parser handles spec with only draft children', () => {
      // Create a new spec with only draft children
      fs.mkdirSync('specs/draft-spec');
      fs.mkdirSync('specs/draft-spec/assertions');
      
      fs.writeFileSync('specs/draft-spec/draft-spec.md', 
`---
id: draft-spec
created: 2026-01-01T00:00:00Z
priority: 1
status: in_progress
---

# Draft Spec

This spec has only draft assertions.
`);
      
      fs.writeFileSync('specs/draft-spec/assertions/draft-assertion.md', 
`---
id: draft-assertion
parent: draft-spec
created: 2026-01-01T00:00:00Z
priority: 1
status: draft
---

# Draft Assertion

This is a draft assertion.
`);
      
      // Parse specs
      const { specs } = parseAllSpecs();
      
      // Find the draft spec
      const draftSpec = specs.find(s => s.id === 'draft-spec');
      assert.ok(draftSpec, 'Draft spec should be found');
      assert.strictEqual(draftSpec.status, 'not_started', 
        'Parent status should be not_started when only draft children exist');
    });

  } finally {
    // Clean up
    process.chdir(originalCwd);
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true });
    }
  }
});