import { test, describe } from 'node:test';
import assert from 'node:assert';
import { parseFrontmatter, validateFields } from '../index.js';

describe('Branch Field Validation', () => {
  test('accepts valid branch names', () => {
    const validBranches = [
      'main',
      'feature/chat-system',
      'bugfix/parser-fix',
      'hotfix/critical-bug',
      'feature/my_awesome-branch'
    ];
    
    validBranches.forEach(branch => {
      const data = {
        id: 'test-assertion',
        parent: 'test-spec',
        created: '2026-01-20T16:00:00Z',
        priority: 1,
        branch
      };
      
      assert.doesNotThrow(() => validateFields(data, 'test.md', true),
        `Should accept valid branch: ${branch}`);
    });
  });
  
  test('rejects invalid branch names', () => {
    const invalidCases = [
      { branch: 'feature/chat system', error: /invalid characters/ },
      { branch: 123, error: /must be a string/ },
      { branch: '/feature/test', error: /cannot start or end with/ },
      { branch: 'feature/test/', error: /cannot start or end with/ },
      { branch: 'feature/test@branch', error: /invalid characters/ }
    ];
    
    invalidCases.forEach(({ branch, error }) => {
      const data = {
        id: 'test-assertion',
        parent: 'test-spec',
        created: '2026-01-20T16:00:00Z',
        priority: 1,
        branch
      };
      
      assert.throws(() => validateFields(data, 'test.md', true), error,
        `Should reject invalid branch: ${branch}`);
    });
  });
  
  test('warns for non-standard branch patterns but allows omitted field', () => {
    const originalWarn = console.warn;
    const warnings = [];
    console.warn = (msg) => warnings.push(msg);
    
    // Non-standard pattern should warn
    const nonStandardData = {
      id: 'test-assertion',
      parent: 'test-spec',
      created: '2026-01-20T16:00:00Z',
      priority: 1,
      branch: 'my-custom-branch'
    };
    
    assert.doesNotThrow(() => validateFields(nonStandardData, 'test.md', true));
    assert(warnings.length > 0);
    assert(warnings[0].includes('non-standard pattern'));
    
    // Omitted field should not throw
    warnings.length = 0;
    const omittedData = {
      id: 'test-assertion',
      parent: 'test-spec',
      created: '2026-01-20T16:00:00Z',
      priority: 1
    };
    
    assert.doesNotThrow(() => validateFields(omittedData, 'test.md', true));
    assert.strictEqual(warnings.length, 0);
    
    console.warn = originalWarn;
  });
  
  test('extracts branch field from YAML frontmatter', () => {
    const content = `---
id: test-assertion
parent: test-spec
created: 2026-01-20T16:00:00Z
priority: 1
branch: feature/chat-system
---

# Test Assertion`;
    
    const { data } = parseFrontmatter(content);
    assert.strictEqual(data.branch, 'feature/chat-system');
  });
});
