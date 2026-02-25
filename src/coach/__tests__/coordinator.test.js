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
  });

  afterEach(() => {
    // Clean up test directory
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }
  });

  // Helper function to initialize git repo for tests that need it
  function initGitRepo(dir) {
    execSync('git init', { cwd: dir, stdio: 'pipe' });
    execSync('git config user.email "test@example.com"', { cwd: dir, stdio: 'pipe' });
    execSync('git config user.name "Test User"', { cwd: dir, stdio: 'pipe' });
    fs.writeFileSync(path.join(dir, 'README.md'), '# Test\n');
    execSync('git add README.md', { cwd: dir, stdio: 'pipe' });
    execSync('git commit -m "Initial commit"', { cwd: dir, stdio: 'pipe' });
    try {
      execSync('git branch -M main', { cwd: dir, stdio: 'pipe' });
    } catch (e) {
      // Branch already named main
    }
  }

  describe('Skill Interface', () => {
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

  describe('Analyze Dependencies (LLM-based)', () => {
    it('should return dependency map', async () => {
      const assertions = [
        { id: 'assertion-a', content: 'First assertion', parent: 'test-spec' },
        { id: 'assertion-b', content: 'Second assertion depends on assertion-a', parent: 'test-spec' }
      ];

      const dependencies = await coordinator.analyzeDependencies(assertions);
      
      assert.ok('assertion-a' in dependencies);
      assert.ok('assertion-b' in dependencies);
      assert.strictEqual(dependencies['assertion-b']['depends-on'], 'assertion-a');
    });

    it('should detect no dependency when no references found', async () => {
      const assertions = [
        { id: 'assertion-a', content: 'First assertion', parent: 'test-spec' },
        { id: 'assertion-b', content: 'Second assertion', parent: 'test-spec' }
      ];

      const dependencies = await coordinator.analyzeDependencies(assertions);
      
      assert.strictEqual(dependencies['assertion-a']['depends-on'], null);
      assert.strictEqual(dependencies['assertion-b']['depends-on'], null);
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

  describe('Update Assertion Frontmatter', () => {
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

    it('should add both depends-on and branch fields', () => {
      coordinator.updateAssertionFrontmatter(assertionFile, {
        'depends-on': 'parent-assertion',
        branch: 'feature/test-spec'
      });
      
      const content = fs.readFileSync(assertionFile, 'utf8');
      assert.ok(content.includes('depends-on: parent-assertion'));
      assert.ok(content.includes('branch: feature/test-spec'));
    });

    it('should add only branch field when depends-on is not provided', () => {
      coordinator.updateAssertionFrontmatter(assertionFile, {
        branch: 'feature/test-spec'
      });
      
      const content = fs.readFileSync(assertionFile, 'utf8');
      assert.ok(!content.includes('depends-on:'));
      assert.ok(content.includes('branch: feature/test-spec'));
    });

    it('should update existing fields', () => {
      // First add fields
      coordinator.updateAssertionFrontmatter(assertionFile, {
        'depends-on': 'first-parent',
        branch: 'feature/old'
      });
      
      // Then update them
      coordinator.updateAssertionFrontmatter(assertionFile, {
        'depends-on': 'second-parent',
        branch: 'feature/new'
      });
      
      const content = fs.readFileSync(assertionFile, 'utf8');
      assert.ok(content.includes('depends-on: second-parent'));
      assert.ok(content.includes('branch: feature/new'));
      assert.ok(!content.includes('first-parent'));
      assert.ok(!content.includes('feature/old'));
    });

    it('should preserve existing frontmatter fields', () => {
      coordinator.updateAssertionFrontmatter(assertionFile, {
        branch: 'feature/test'
      });
      
      const content = fs.readFileSync(assertionFile, 'utf8');
      assert.ok(content.includes('id: test-assertion'));
      assert.ok(content.includes('parent: test-spec'));
      assert.ok(content.includes('status: not_started'));
      assert.ok(content.includes('priority: 1'));
    });

    it('should preserve markdown content', () => {
      coordinator.updateAssertionFrontmatter(assertionFile, {
        branch: 'feature/test'
      });
      
      const content = fs.readFileSync(assertionFile, 'utf8');
      assert.ok(content.includes('# Test Assertion'));
      assert.ok(content.includes('Content here.'));
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

  describe('Run YAML Frontmatter Updates', () => {
    beforeEach(() => {
      initGitRepo(testDir);
      // Create test spec structure
      const specsDir = path.join(testDir, 'specs');
      
      // Spec 1 with connected assertions
      const spec1 = path.join(specsDir, 'spec-1');
      const assertions1 = path.join(spec1, 'assertions');
      fs.mkdirSync(assertions1, { recursive: true });

      fs.writeFileSync(
        path.join(spec1, 'spec-1.md'),
        `---
id: spec-1
created: 2026-02-01T00:00:00Z
priority: 1
---

# Spec 1
`
      );

      fs.writeFileSync(
        path.join(assertions1, 'assertion-a.md'),
        `---
id: assertion-a
parent: spec-1
created: 2026-02-01T00:00:00Z
priority: 1
status: not_started
---

# Assertion A
`
      );

      fs.writeFileSync(
        path.join(assertions1, 'assertion-b.md'),
        `---
id: assertion-b
parent: spec-1
created: 2026-02-01T00:00:00Z
priority: 1
status: not_started
---

# Assertion B depends on assertion-a
`
      );

      // Spec 2 with isolated assertion
      const spec2 = path.join(specsDir, 'spec-2');
      const assertions2 = path.join(spec2, 'assertions');
      fs.mkdirSync(assertions2, { recursive: true });

      fs.writeFileSync(
        path.join(spec2, 'spec-2.md'),
        `---
id: spec-2
created: 2026-02-01T00:00:00Z
priority: 1
---

# Spec 2
`
      );

      fs.writeFileSync(
        path.join(assertions2, 'isolated.md'),
        `---
id: isolated
parent: spec-2
created: 2026-02-01T00:00:00Z
priority: 1
status: not_started
---

# Isolated Assertion
`
      );
    });

    it('should update all assertions with dependency and branch metadata', async () => {
      const result = await coordinator.runYAMLFrontmatterUpdates(testDir);
      
      assert.strictEqual(result.success, true);
      assert.ok(result.assertionCount > 0);
      assert.ok(result.filesUpdated > 0);
      assert.ok(result.commitHash);
      
      // Verify files were updated
      const contentA = fs.readFileSync(
        path.join(testDir, 'specs/spec-1/assertions/assertion-a.md'),
        'utf8'
      );
      const contentB = fs.readFileSync(
        path.join(testDir, 'specs/spec-1/assertions/assertion-b.md'),
        'utf8'
      );
      const contentIsolated = fs.readFileSync(
        path.join(testDir, 'specs/spec-2/assertions/isolated.md'),
        'utf8'
      );
      
      // Check assertion-a (part of multi-assertion spec)
      assert.ok(contentA.includes('branch: feature/spec-1'));
      assert.ok(!contentA.includes('depends-on:'));
      
      // Check assertion-b (depends on assertion-a)
      assert.ok(contentB.includes('branch: feature/spec-1'));
      assert.ok(contentB.includes('depends-on: assertion-a'));
      
      // Check isolated assertion
      assert.ok(contentIsolated.includes('branch: main'));
      assert.ok(!contentIsolated.includes('depends-on:'));
    });

    it('should support dry-run mode', async () => {
      const result = await coordinator.runYAMLFrontmatterUpdates(testDir, { dryRun: true });
      
      assert.strictEqual(result.success, true);
      assert.strictEqual(result.dryRun, true);
      assert.ok(result.preview);
      
      // Verify files were NOT updated
      const contentA = fs.readFileSync(
        path.join(testDir, 'specs/spec-1/assertions/assertion-a.md'),
        'utf8'
      );
      
      assert.ok(!contentA.includes('branch:'));
    });

    it('should handle no assertions gracefully', async () => {
      // Remove all assertions
      const specsDir = path.join(testDir, 'specs');
      fs.rmSync(specsDir, { recursive: true, force: true });
      fs.mkdirSync(specsDir, { recursive: true });

      const result = await coordinator.runYAMLFrontmatterUpdates(testDir);
      
      assert.strictEqual(result.success, true);
      assert.strictEqual(result.assertionCount, 0);
    });
  });

  describe('Assign Branches', () => {
    it('should assign feature branch for multi-assertion specs', () => {
      const assertions = [
        { id: 'assertion-a', parent: 'spec-1' },
        { id: 'assertion-b', parent: 'spec-1' }
      ];
      const dependencies = {
        'assertion-a': { 'depends-on': null },
        'assertion-b': { 'depends-on': 'assertion-a' }
      };

      const assignments = coordinator.assignBranches(assertions, dependencies);
      
      assert.strictEqual(assignments.length, 1);
      assert.strictEqual(assignments[0].branch, 'feature/spec-1');
      assert.strictEqual(assignments[0].assertions.length, 2);
    });

    it('should assign main branch for isolated assertions', () => {
      const assertions = [
        { id: 'isolated', parent: 'spec-1' }
      ];
      const dependencies = {
        'isolated': { 'depends-on': null }
      };

      const assignments = coordinator.assignBranches(assertions, dependencies);
      
      assert.strictEqual(assignments.length, 1);
      assert.strictEqual(assignments[0].branch, 'main');
      assert.strictEqual(assignments[0].isIsolated, true);
    });
  });

  describe('Build Preview', () => {
    it('should show dependencies in preview', () => {
      const branchAssignments = [
        {
          branch: 'feature/spec-1',
          assertions: [
            { id: 'assertion-a' },
            { id: 'assertion-b' }
          ]
        }
      ];
      const dependencies = {
        'assertion-a': { 'depends-on': null },
        'assertion-b': { 'depends-on': 'assertion-a' }
      };

      const preview = coordinator.buildPreview(branchAssignments, dependencies);
      
      assert.ok(preview.includes('feature/spec-1'));
      assert.ok(preview.includes('assertion-a'));
      assert.ok(preview.includes('assertion-b'));
      assert.ok(preview.includes('depends on assertion-a'));
    });
  });

  describe('Build Commit Message', () => {
    it('should build comprehensive commit message', () => {
      const branchAssignments = [
        {
          branch: 'feature/test-spec',
          assertions: [
            { id: 'assertion-a' },
            { id: 'assertion-b' }
          ]
        }
      ];
      const dependencies = {
        'assertion-a': { 'depends-on': null },
        'assertion-b': { 'depends-on': 'assertion-a' }
      };

      const message = coordinator.buildCommitMessage(branchAssignments, dependencies, 2);
      
      assert.ok(message.includes('Add coordinator dependency and branch metadata'));
      assert.ok(message.includes('feature/test-spec'));
      assert.ok(message.includes('assertion-a'));
      assert.ok(message.includes('assertion-b'));
      assert.ok(message.includes('Added depends-on field where dependencies exist'));
      assert.ok(message.includes('Added branch field to all assertions'));
    });
  });
});
