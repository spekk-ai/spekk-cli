import { describe, it, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert';
import { Coordinator } from '../coordinator.js';
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import { execSync } from 'node:child_process';

describe('Coordinator Skill', () => {
  let coordinator;
  let testDir;

  beforeEach(() => {
    coordinator = new Coordinator();
    // Create a temporary test directory
    testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'coordinator-test-'));
    
    // Initialize git repo
    execSync('git init', { cwd: testDir, stdio: 'pipe' });
    execSync('git config user.email "test@example.com"', { cwd: testDir, stdio: 'pipe' });
    execSync('git config user.name "Test User"', { cwd: testDir, stdio: 'pipe' });
  });

  afterEach(() => {
    // Clean up test directory
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }
  });

  describe('Skill Interface', () => {
    it('should have correct ID', () => {
      assert.strictEqual(coordinator.getId(), 'coordinator');
    });

    it('should have correct name', () => {
      assert.strictEqual(coordinator.getName(), 'Coordinator');
    });

    it('should have description', () => {
      const desc = coordinator.getDescription();
      assert.ok(desc);
      assert.ok(desc.toLowerCase().includes('dependencies'));
    });

    it('should trigger on coordinator keywords', () => {
      assert.strictEqual(coordinator.shouldTrigger('run coordinator'), true);
      assert.strictEqual(coordinator.shouldTrigger('analyze dependencies'), true);
      assert.strictEqual(coordinator.shouldTrigger('dependency analysis'), true);
      assert.strictEqual(coordinator.shouldTrigger('plan work'), true);
    });

    it('should not trigger on unrelated input', () => {
      assert.strictEqual(coordinator.shouldTrigger('hello world'), false);
      assert.strictEqual(coordinator.shouldTrigger('build something'), false);
    });

    it('should have questions', () => {
      const questions = coordinator.getQuestions();
      assert.ok(Array.isArray(questions));
      assert.ok(questions.length > 0);
    });
  });

  describe('Read Draft Assertions', () => {
    beforeEach(() => {
      // Create test spec structure
      const specsDir = path.join(testDir, 'specs');
      const testSpec = path.join(specsDir, 'test-spec');
      const assertionsDir = path.join(testSpec, 'assertions');
      
      fs.mkdirSync(assertionsDir, { recursive: true });

      // Create parent spec
      fs.writeFileSync(
        path.join(testSpec, 'test-spec.md'),
        `---
id: test-spec
created: 2026-02-01T00:00:00Z
priority: 1
---

# Test Spec

Test parent spec.
`
      );

      // Create draft assertion
      fs.writeFileSync(
        path.join(assertionsDir, 'draft-assertion.md'),
        `---
id: draft-assertion
parent: test-spec
created: 2026-02-01T00:00:00Z
priority: 1
status: draft
---

# Draft Assertion

This is a draft assertion.
`
      );

      // Create not_started assertion
      fs.writeFileSync(
        path.join(assertionsDir, 'not-started-assertion.md'),
        `---
id: not-started-assertion
parent: test-spec
created: 2026-02-01T00:00:00Z
priority: 1
status: not_started
---

# Not Started Assertion

This is a not started assertion.
`
      );

      // Create done assertion (should be excluded)
      fs.writeFileSync(
        path.join(assertionsDir, 'done-assertion.md'),
        `---
id: done-assertion
parent: test-spec
created: 2026-02-01T00:00:00Z
priority: 1
status: done
---

# Done Assertion

This is done.
`
      );
    });

    it('should read only draft and not_started assertions', () => {
      const assertions = coordinator.readDraftAssertions(testDir);
      
      assert.strictEqual(assertions.length, 2);
      assert.ok(assertions.some(a => a.id === 'draft-assertion'));
      assert.ok(assertions.some(a => a.id === 'not-started-assertion'));
      assert.ok(!assertions.some(a => a.id === 'done-assertion'));
    });

    it('should include required fields', () => {
      const assertions = coordinator.readDraftAssertions(testDir);
      const assertion = assertions[0];
      
      assert.ok('id' in assertion);
      assert.ok('parent' in assertion);
      assert.ok('filePath' in assertion);
      assert.ok('status' in assertion);
      assert.ok('priority' in assertion);
    });
  });

  describe('Analyze Dependencies', () => {
    it('should return dependency map', () => {
      const assertions = [
        { id: 'assertion-a', content: 'First assertion', parent: 'test-spec' },
        { id: 'assertion-b', content: 'Second assertion depends on assertion-a', parent: 'test-spec' }
      ];

      const dependencies = coordinator.analyzeDependencies(assertions);
      
      assert.ok('assertion-a' in dependencies);
      assert.ok('assertion-b' in dependencies);
      assert.strictEqual(dependencies['assertion-b']['depends-on'], 'assertion-a');
    });

    it('should detect no dependency when no references found', () => {
      const assertions = [
        { id: 'assertion-a', content: 'First assertion', parent: 'test-spec' },
        { id: 'assertion-b', content: 'Second assertion', parent: 'test-spec' }
      ];

      const dependencies = coordinator.analyzeDependencies(assertions);
      
      assert.strictEqual(dependencies['assertion-a']['depends-on'], null);
      assert.strictEqual(dependencies['assertion-b']['depends-on'], null);
    });

    it('should detect domain dependencies', () => {
      const assertions = [
        { id: 'session-model', title: 'Session Model', content: 'Model for sessions', parent: 'test-spec' },
        { id: 'message-input', title: 'Message Input', content: 'Input for messages', parent: 'test-spec' }
      ];

      const dependencies = coordinator.analyzeDependencies(assertions);
      
      // message-input should depend on session-model
      assert.strictEqual(dependencies['message-input']['depends-on'], 'session-model');
    });
  });

  describe('Circular Dependency Detection', () => {
    it('should detect circular dependencies', () => {
      const dependencies = {
        'assertion-a': { 'depends-on': 'assertion-b' },
        'assertion-b': { 'depends-on': 'assertion-a' }
      };

      assert.throws(() => {
        coordinator.detectCircularDependencies(dependencies);
      }, /Circular dependency detected/);
    });

    it('should detect longer circular chains', () => {
      const dependencies = {
        'assertion-a': { 'depends-on': 'assertion-b' },
        'assertion-b': { 'depends-on': 'assertion-c' },
        'assertion-c': { 'depends-on': 'assertion-a' }
      };

      assert.throws(() => {
        coordinator.detectCircularDependencies(dependencies);
      }, /Circular dependency detected/);
    });

    it('should allow valid dependency chains', () => {
      const dependencies = {
        'assertion-a': { 'depends-on': null },
        'assertion-b': { 'depends-on': 'assertion-a' },
        'assertion-c': { 'depends-on': 'assertion-b' }
      };

      assert.doesNotThrow(() => {
        coordinator.detectCircularDependencies(dependencies);
      });
    });
  });

  describe('Build Dependency Tree', () => {
    it('should build visual tree representation', () => {
      const assertions = [
        { id: 'assertion-a', parent: 'test-spec' },
        { id: 'assertion-b', parent: 'test-spec' }
      ];
      const dependencies = {
        'assertion-a': { 'depends-on': null },
        'assertion-b': { 'depends-on': 'assertion-a' }
      };

      const tree = coordinator.buildDependencyTree(assertions, dependencies);
      
      assert.ok(tree.includes('Dependency Analysis'));
      assert.ok(tree.includes('test-spec:'));
      assert.ok(tree.includes('assertion-a'));
      assert.ok(tree.includes('assertion-b'));
    });

    it('should group by parent spec', () => {
      const assertions = [
        { id: 'assertion-a', parent: 'spec-1' },
        { id: 'assertion-b', parent: 'spec-2' }
      ];
      const dependencies = {
        'assertion-a': { 'depends-on': null },
        'assertion-b': { 'depends-on': null }
      };

      const tree = coordinator.buildDependencyTree(assertions, dependencies);
      
      assert.ok(tree.includes('spec-1:'));
      assert.ok(tree.includes('spec-2:'));
    });
  });

  describe('Update Depends-On Field', () => {
    let assertionFile;

    beforeEach(() => {
      assertionFile = path.join(testDir, 'test-assertion.md');
      fs.writeFileSync(
        assertionFile,
        `---
id: test-assertion
parent: test-spec
created: 2026-02-01T00:00:00Z
priority: 1
status: not_started
---

# Test Assertion

Content here.
`
      );
    });

    it('should add depends-on field when not present', () => {
      coordinator.updateDependsOnField(assertionFile, 'parent-assertion');
      
      const content = fs.readFileSync(assertionFile, 'utf8');
      assert.ok(content.includes('depends-on: parent-assertion'));
    });

    it('should update existing depends-on field', () => {
      // First add a depends-on field
      coordinator.updateDependsOnField(assertionFile, 'first-parent');
      
      // Then update it
      coordinator.updateDependsOnField(assertionFile, 'second-parent');
      
      const content = fs.readFileSync(assertionFile, 'utf8');
      assert.ok(content.includes('depends-on: second-parent'));
      assert.ok(!content.includes('first-parent'));
    });

    it('should remove depends-on field when null provided', () => {
      // First add a depends-on field
      coordinator.updateDependsOnField(assertionFile, 'parent-assertion');
      
      // Then remove it
      coordinator.updateDependsOnField(assertionFile, null);
      
      const content = fs.readFileSync(assertionFile, 'utf8');
      assert.ok(!content.includes('depends-on:'));
    });

    it('should preserve other frontmatter fields', () => {
      coordinator.updateDependsOnField(assertionFile, 'parent-assertion');
      
      const content = fs.readFileSync(assertionFile, 'utf8');
      assert.ok(content.includes('id: test-assertion'));
      assert.ok(content.includes('parent: test-spec'));
      assert.ok(content.includes('status: not_started'));
      assert.ok(content.includes('priority: 1'));
    });
  });

  describe('Update Assertion Files', () => {
    beforeEach(() => {
      // Create test spec structure
      const specsDir = path.join(testDir, 'specs');
      const testSpec = path.join(specsDir, 'test-spec');
      const assertionsDir = path.join(testSpec, 'assertions');
      
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(
        path.join(assertionsDir, 'assertion-a.md'),
        `---
id: assertion-a
parent: test-spec
created: 2026-02-01T00:00:00Z
priority: 1
status: not_started
---

# Assertion A
`
      );

      fs.writeFileSync(
        path.join(assertionsDir, 'assertion-b.md'),
        `---
id: assertion-b
parent: test-spec
created: 2026-02-01T00:00:00Z
priority: 1
status: not_started
---

# Assertion B
`
      );
    });

    it('should update multiple assertion files', () => {
      const assertions = [
        { id: 'assertion-a', filePath: path.join(testDir, 'specs/test-spec/assertions/assertion-a.md') },
        { id: 'assertion-b', filePath: path.join(testDir, 'specs/test-spec/assertions/assertion-b.md') }
      ];
      const dependencies = {
        'assertion-a': { 'depends-on': null },
        'assertion-b': { 'depends-on': 'assertion-a' }
      };

      const updatedFiles = coordinator.updateAssertionFiles(dependencies, assertions);
      
      assert.strictEqual(updatedFiles.length, 2);
      
      const contentB = fs.readFileSync(assertions[1].filePath, 'utf8');
      assert.ok(contentB.includes('depends-on: assertion-a'));
    });
  });

  describe('Run Dependency Analysis', () => {
    beforeEach(() => {
      // Create test spec structure
      const specsDir = path.join(testDir, 'specs');
      const testSpec = path.join(specsDir, 'test-spec');
      const assertionsDir = path.join(testSpec, 'assertions');
      
      fs.mkdirSync(assertionsDir, { recursive: true });

      fs.writeFileSync(
        path.join(testSpec, 'test-spec.md'),
        `---
id: test-spec
created: 2026-02-01T00:00:00Z
priority: 1
---

# Test Spec
`
      );

      fs.writeFileSync(
        path.join(assertionsDir, 'assertion-a.md'),
        `---
id: assertion-a
parent: test-spec
created: 2026-02-01T00:00:00Z
priority: 1
status: not_started
---

# Assertion A
`
      );
    });

    it('should run full dependency analysis', async () => {
      const result = await coordinator.runDependencyAnalysis(testDir);
      
      assert.strictEqual(result.success, true);
      assert.ok(result.assertionCount > 0);
      assert.ok('dependencies' in result);
      assert.ok('tree' in result);
      assert.ok('assertions' in result);
    });

    it('should handle no assertions gracefully', async () => {
      // Remove all assertions
      const assertionsDir = path.join(testDir, 'specs/test-spec/assertions');
      fs.rmSync(assertionsDir, { recursive: true, force: true });
      fs.mkdirSync(assertionsDir, { recursive: true });

      const result = await coordinator.runDependencyAnalysis(testDir);
      
      assert.strictEqual(result.success, true);
      assert.strictEqual(result.assertionCount, 0);
    });
  });
});
