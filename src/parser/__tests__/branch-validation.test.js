import { test, describe } from 'node:test';
import assert from 'node:assert';
import { parseFrontmatter, validateFields } from '../index.js';

describe('Branch Field Validation', () => {
  describe('Valid branch names', () => {
    test('accepts standard main branch', () => {
      const data = {
        id: 'test-assertion',
        parent: 'test-spec',
        created: '2026-01-20T16:00:00Z',
        priority: 1,
        branch: 'main'
      };
      
      assert.doesNotThrow(() => validateFields(data, 'test.md', true));
    });
    
    test('accepts feature/ prefix', () => {
      const data = {
        id: 'test-assertion',
        parent: 'test-spec',
        created: '2026-01-20T16:00:00Z',
        priority: 1,
        branch: 'feature/chat-system'
      };
      
      assert.doesNotThrow(() => validateFields(data, 'test.md', true));
    });
    
    test('accepts bugfix/ prefix', () => {
      const data = {
        id: 'test-assertion',
        parent: 'test-spec',
        created: '2026-01-20T16:00:00Z',
        priority: 1,
        branch: 'bugfix/parser-fix'
      };
      
      assert.doesNotThrow(() => validateFields(data, 'test.md', true));
    });
    
    test('accepts hotfix/ prefix', () => {
      const data = {
        id: 'test-assertion',
        parent: 'test-spec',
        created: '2026-01-20T16:00:00Z',
        priority: 1,
        branch: 'hotfix/critical-bug'
      };
      
      assert.doesNotThrow(() => validateFields(data, 'test.md', true));
    });
    
    test('accepts branch with hyphens and underscores', () => {
      const data = {
        id: 'test-assertion',
        parent: 'test-spec',
        created: '2026-01-20T16:00:00Z',
        priority: 1,
        branch: 'feature/my_awesome-branch'
      };
      
      assert.doesNotThrow(() => validateFields(data, 'test.md', true));
    });
  });
  
  describe('Invalid branch names', () => {
    test('rejects branch with spaces', () => {
      const data = {
        id: 'test-assertion',
        parent: 'test-spec',
        created: '2026-01-20T16:00:00Z',
        priority: 1,
        branch: 'feature/chat system'
      };
      
      assert.throws(() => validateFields(data, 'test.md', true), {
        message: /Field 'branch' contains invalid characters/
      });
    });
    
    test('rejects non-string branch', () => {
      const data = {
        id: 'test-assertion',
        parent: 'test-spec',
        created: '2026-01-20T16:00:00Z',
        priority: 1,
        branch: 123
      };
      
      assert.throws(() => validateFields(data, 'test.md', true), {
        message: /Field 'branch' must be a string/
      });
    });
    
    test('rejects branch starting with slash', () => {
      const data = {
        id: 'test-assertion',
        parent: 'test-spec',
        created: '2026-01-20T16:00:00Z',
        priority: 1,
        branch: '/feature/test'
      };
      
      assert.throws(() => validateFields(data, 'test.md', true), {
        message: /Field 'branch' cannot start or end with/
      });
    });
    
    test('rejects branch ending with slash', () => {
      const data = {
        id: 'test-assertion',
        parent: 'test-spec',
        created: '2026-01-20T16:00:00Z',
        priority: 1,
        branch: 'feature/test/'
      };
      
      assert.throws(() => validateFields(data, 'test.md', true), {
        message: /Field 'branch' cannot start or end with/
      });
    });
    
    test('rejects branch with special characters', () => {
      const data = {
        id: 'test-assertion',
        parent: 'test-spec',
        created: '2026-01-20T16:00:00Z',
        priority: 1,
        branch: 'feature/test@branch'
      };
      
      assert.throws(() => validateFields(data, 'test.md', true), {
        message: /Field 'branch' contains invalid characters/
      });
    });
  });
  
  describe('Branch field warnings', () => {
    test('warns for non-standard branch patterns', () => {
      const data = {
        id: 'test-assertion',
        parent: 'test-spec',
        created: '2026-01-20T16:00:00Z',
        priority: 1,
        branch: 'my-custom-branch'
      };
      
      const originalWarn = console.warn;
      const warnings = [];
      console.warn = (msg) => warnings.push(msg);
      
      assert.doesNotThrow(() => validateFields(data, 'test.md', true));
      assert(warnings.length > 0);
      assert(warnings[0].includes('non-standard pattern'));
      
      console.warn = originalWarn;
    });
    
    test('does not warn for standard patterns', () => {
      const standardBranches = ['main', 'master', 'develop', 'feature/x', 'bugfix/y', 'hotfix/z'];
      
      const originalWarn = console.warn;
      const warnings = [];
      console.warn = (msg) => warnings.push(msg);
      
      for (const branch of standardBranches) {
        const data = {
          id: 'test-assertion',
          parent: 'test-spec',
          created: '2026-01-20T16:00:00Z',
          priority: 1,
          branch
        };
        validateFields(data, 'test.md', true);
      }
      
      assert.strictEqual(warnings.length, 0);
      console.warn = originalWarn;
    });
  });
  
  describe('Omitted branch field', () => {
    test('allows omitted branch field', () => {
      const data = {
        id: 'test-assertion',
        parent: 'test-spec',
        created: '2026-01-20T16:00:00Z',
        priority: 1
      };
      
      assert.doesNotThrow(() => validateFields(data, 'test.md', true));
    });
  });
  
  describe('Parsing branch from frontmatter', () => {
    test('extracts branch field from YAML', () => {
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
});
