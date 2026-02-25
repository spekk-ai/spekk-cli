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
    
    // Create initial commit so we have a proper branch
    fs.writeFileSync(path.join(testDir, 'README.md'), '# Test\n');
    execSync('git add README.md', { cwd: testDir, stdio: 'pipe' });
    execSync('git commit -m "Initial commit"', { cwd: testDir, stdio: 'pipe' });
    
    // Ensure we're on main branch (git init might create main or master)
    try {
      execSync('git branch -M main', { cwd: testDir, stdio: 'pipe' });
    } catch (e) {
      // Branch already named main
    }
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

  describe('Identify Dependency Clusters', () => {
    it('should identify single cluster for connected assertions', () => {
      const assertions = [
        { id: 'assertion-a', parent: 'test-spec' },
        { id: 'assertion-b', parent: 'test-spec' },
        { id: 'assertion-c', parent: 'test-spec' }
      ];
      const dependencies = {
        'assertion-a': { 'depends-on': null },
        'assertion-b': { 'depends-on': 'assertion-a' },
        'assertion-c': { 'depends-on': 'assertion-b' }
      };

      const clusters = coordinator.identifyDependencyClusters(assertions, dependencies);
      
      assert.strictEqual(clusters.length, 1);
      assert.strictEqual(clusters[0].length, 3);
    });

    it('should identify multiple clusters for independent assertions', () => {
      const assertions = [
        { id: 'assertion-a', parent: 'spec-1' },
        { id: 'assertion-b', parent: 'spec-1' },
        { id: 'assertion-c', parent: 'spec-2' },
        { id: 'assertion-d', parent: 'spec-2' }
      ];
      const dependencies = {
        'assertion-a': { 'depends-on': null },
        'assertion-b': { 'depends-on': 'assertion-a' },
        'assertion-c': { 'depends-on': null },
        'assertion-d': { 'depends-on': 'assertion-c' }
      };

      const clusters = coordinator.identifyDependencyClusters(assertions, dependencies);
      
      assert.strictEqual(clusters.length, 2);
    });

    it('should handle isolated assertions', () => {
      const assertions = [
        { id: 'assertion-a', parent: 'test-spec' },
        { id: 'assertion-b', parent: 'test-spec' }
      ];
      const dependencies = {
        'assertion-a': { 'depends-on': null },
        'assertion-b': { 'depends-on': null }
      };

      const clusters = coordinator.identifyDependencyClusters(assertions, dependencies);
      
      assert.strictEqual(clusters.length, 2);
      assert.strictEqual(clusters[0].length, 1);
      assert.strictEqual(clusters[1].length, 1);
    });
  });

  describe('Generate Branch Name', () => {
    it('should use parent spec ID for single-parent cluster', () => {
      const cluster = [
        { id: 'assertion-a', parent: 'test-spec' },
        { id: 'assertion-b', parent: 'test-spec' }
      ];

      const branchName = coordinator.generateBranchName(cluster);
      
      assert.strictEqual(branchName, 'feature/test-spec');
    });

    it('should use most common parent for multi-parent cluster', () => {
      const cluster = [
        { id: 'assertion-a', parent: 'spec-1' },
        { id: 'assertion-b', parent: 'spec-1' },
        { id: 'assertion-c', parent: 'spec-2' }
      ];

      const branchName = coordinator.generateBranchName(cluster);
      
      assert.strictEqual(branchName, 'feature/spec-1');
    });
  });

  describe('Assign Branches to Clusters', () => {
    it('should assign feature branches to connected clusters', () => {
      const clusters = [
        [
          { id: 'assertion-a', parent: 'spec-1' },
          { id: 'assertion-b', parent: 'spec-1' }
        ],
        [
          { id: 'assertion-c', parent: 'spec-2' },
          { id: 'assertion-d', parent: 'spec-2' }
        ]
      ];

      const assignments = coordinator.assignBranchesToClusters(clusters);
      
      assert.ok(assignments.some(a => a.branch === 'feature/spec-1'));
      assert.ok(assignments.some(a => a.branch === 'feature/spec-2'));
    });

    it('should assign isolated assertions to main branch', () => {
      const clusters = [
        [{ id: 'isolated-1', parent: 'spec-1' }],
        [{ id: 'isolated-2', parent: 'spec-2' }]
      ];

      const assignments = coordinator.assignBranchesToClusters(clusters);
      
      const mainBranch = assignments.find(a => a.branch === 'main');
      assert.ok(mainBranch);
      assert.strictEqual(mainBranch.assertions.length, 2);
    });

    it('should combine all isolated assertions into single main group', () => {
      const clusters = [
        [{ id: 'isolated-1', parent: 'spec-1' }],
        [{ id: 'isolated-2', parent: 'spec-2' }],
        [{ id: 'isolated-3', parent: 'spec-3' }]
      ];

      const assignments = coordinator.assignBranchesToClusters(clusters);
      
      assert.strictEqual(assignments.length, 1);
      assert.strictEqual(assignments[0].branch, 'main');
      assert.strictEqual(assignments[0].assertions.length, 3);
    });
  });

  describe('Format Branch Assignments', () => {
    it('should format branch assignments with counts', () => {
      const assignments = [
        {
          branch: 'feature/spec-1',
          assertions: [
            { id: 'assertion-a' },
            { id: 'assertion-b' }
          ],
          isIsolated: false
        },
        {
          branch: 'main',
          assertions: [
            { id: 'isolated-1' }
          ],
          isIsolated: false
        }
      ];

      const output = coordinator.formatBranchAssignments(assignments);
      
      assert.ok(output.includes('Branch Assignments'));
      assert.ok(output.includes('feature/spec-1 (2 assertions)'));
      assert.ok(output.includes('assertion-a'));
      assert.ok(output.includes('assertion-b'));
      assert.ok(output.includes('main (1 isolated assertion)'));
    });

    it('should sort with feature branches first, main last', () => {
      const assignments = [
        { branch: 'main', assertions: [{ id: 'isolated' }], isIsolated: false },
        { branch: 'feature/spec-b', assertions: [{ id: 'b1' }], isIsolated: false },
        { branch: 'feature/spec-a', assertions: [{ id: 'a1' }], isIsolated: false }
      ];

      const output = coordinator.formatBranchAssignments(assignments);
      
      const mainPos = output.indexOf('main');
      const specAPos = output.indexOf('feature/spec-a');
      const specBPos = output.indexOf('feature/spec-b');
      
      assert.ok(specAPos < mainPos);
      assert.ok(specBPos < mainPos);
    });
  });

  describe('Branch Exists', () => {
    beforeEach(() => {
      // Create a test branch
      execSync('git checkout -b test-branch', { cwd: testDir, stdio: 'pipe' });
      execSync('git checkout main', { cwd: testDir, stdio: 'pipe' });
    });

    it('should detect existing branch', () => {
      const exists = coordinator.branchExists('test-branch', testDir);
      assert.strictEqual(exists, true);
    });

    it('should return false for non-existent branch', () => {
      const exists = coordinator.branchExists('non-existent-branch', testDir);
      assert.strictEqual(exists, false);
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

  describe('Update Assertion Files with Branch Metadata', () => {
    beforeEach(() => {
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

    it('should update multiple files with branch and dependency metadata', () => {
      const branchAssignments = [
        {
          branch: 'feature/test-spec',
          assertions: [
            { id: 'assertion-a', filePath: path.join(testDir, 'specs/test-spec/assertions/assertion-a.md') },
            { id: 'assertion-b', filePath: path.join(testDir, 'specs/test-spec/assertions/assertion-b.md') }
          ]
        }
      ];
      
      const dependencies = {
        'assertion-a': { 'depends-on': null },
        'assertion-b': { 'depends-on': 'assertion-a' }
      };

      const updatedFiles = coordinator.updateAssertionFilesWithBranchMetadata(
        branchAssignments,
        dependencies,
        testDir
      );
      
      assert.strictEqual(updatedFiles.length, 2);
      
      const contentA = fs.readFileSync(branchAssignments[0].assertions[0].filePath, 'utf8');
      const contentB = fs.readFileSync(branchAssignments[0].assertions[1].filePath, 'utf8');
      
      assert.ok(contentA.includes('branch: feature/test-spec'));
      assert.ok(!contentA.includes('depends-on:'));
      
      assert.ok(contentB.includes('branch: feature/test-spec'));
      assert.ok(contentB.includes('depends-on: assertion-a'));
    });
  });

  describe('Format Branch Dependency Tree', () => {
    it('should format simple dependency chain', () => {
      const assertions = [
        { id: 'assertion-a' },
        { id: 'assertion-b' }
      ];
      const dependencies = {
        'assertion-a': { 'depends-on': null },
        'assertion-b': { 'depends-on': 'assertion-a' }
      };

      const tree = coordinator.formatBranchDependencyTree(assertions, dependencies);
      
      assert.ok(tree.includes('assertion-a → assertion-b'));
    });

    it('should format multiple independent assertions', () => {
      const assertions = [
        { id: 'assertion-a' },
        { id: 'assertion-b' }
      ];
      const dependencies = {
        'assertion-a': { 'depends-on': null },
        'assertion-b': { 'depends-on': null }
      };

      const tree = coordinator.formatBranchDependencyTree(assertions, dependencies);
      
      assert.ok(tree.includes('assertion-a'));
      assert.ok(tree.includes('assertion-b'));
      assert.ok(!tree.includes('→'));
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

  describe('Run Branch Assignment', () => {
    beforeEach(() => {
      // Create test spec structure with multiple specs
      const specsDir = path.join(testDir, 'specs');
      
      // Spec 1
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

      // Spec 2 (isolated assertion)
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

    it('should run full branch assignment', async () => {
      const result = await coordinator.runBranchAssignment(testDir);
      
      assert.strictEqual(result.success, true);
      assert.ok(result.assertionCount > 0);
      assert.ok(result.clusterCount > 0);
      assert.ok('branchAssignments' in result);
      assert.ok('output' in result);
    });

    it('should identify feature and main branches', async () => {
      const result = await coordinator.runBranchAssignment(testDir);
      
      const branches = result.branchAssignments.map(a => a.branch);
      assert.ok(branches.includes('feature/spec-1'));
      assert.ok(branches.includes('main'));
    });

    it('should warn about existing branches', async () => {
      // Create a branch that conflicts
      execSync('git checkout -b feature/spec-1', { cwd: testDir, stdio: 'pipe' });
      execSync('git checkout main', { cwd: testDir, stdio: 'pipe' });

      const result = await coordinator.runBranchAssignment(testDir);
      
      assert.ok(result.warnings.length > 0);
      assert.ok(result.warnings.some(w => w.includes('feature/spec-1')));
    });

    it('should handle no assertions gracefully', async () => {
      // Remove all assertions
      const specsDir = path.join(testDir, 'specs');
      fs.rmSync(specsDir, { recursive: true, force: true });
      fs.mkdirSync(specsDir, { recursive: true });

      const result = await coordinator.runBranchAssignment(testDir);
      
      assert.strictEqual(result.success, true);
      assert.strictEqual(result.assertionCount, 0);
    });
  });

  describe('Run YAML Frontmatter Updates', () => {
    beforeEach(() => {
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
      
      // Check assertion-a
      assert.ok(contentA.includes('branch: feature/spec-1'));
      assert.ok(!contentA.includes('depends-on:'));
      
      // Check assertion-b
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
});
