import fs from 'node:fs';
import path from 'node:path';
import { execSync } from 'node:child_process';
import { Skill } from './skill-interface.js';
import { parseAllSpecs } from '../parser/index.js';

/**
 * Coordinator skill
 * Analyzes dependencies between assertions and builds single-parent dependency trees.
 * This is a coach skill that extends planning capabilities.
 */
export class Coordinator extends Skill {
  constructor() {
    super();
    this.triggerKeywords = [
      'coordinator', 'analyze dependencies', 'dependency analysis',
      'plan work', 'build dependency tree', 'coordinate work'
    ];
  }

  getId() {
    return 'coordinator';
  }

  getName() {
    return 'Coordinator';
  }

  getDescription() {
    return 'Analyzes dependencies between assertions and builds single-parent dependency trees for work planning.';
  }

  shouldTrigger(userInput, context = {}) {
    const lowerInput = userInput.toLowerCase();
    return this.triggerKeywords.some(keyword => lowerInput.includes(keyword));
  }

  getQuestions() {
    return [
      {
        id: 'confirmation',
        text: 'Analyze dependencies for draft and not_started assertions?',
        type: 'choice',
        options: ['yes', 'no']
      }
    ];
  }

  processResponses(responses) {
    return {
      summary: 'Dependency analysis initiated',
      recommendations: [
        'Analyze assertion requirements and relationships',
        'Build single-parent dependency tree',
        'Present plan for user confirmation'
      ],
      data: responses
    };
  }

  /**
   * Read all assertions with draft or not_started status
   * @param {string} baseDir - Project root directory
   * @returns {Array<Object>} Array of assertion objects
   */
  readDraftAssertions(baseDir = process.cwd()) {
    const specsDir = path.join(baseDir, 'specs');
    const { assertions } = parseAllSpecs(specsDir);
    
    const draftAssertions = [];
    
    for (const assertion of assertions) {
      if (assertion.status === 'draft' || assertion.status === 'not_started') {
        draftAssertions.push({
          id: assertion.id,
          parent: assertion.parent,
          filePath: path.join(baseDir, assertion.file),
          title: assertion.title,
          content: assertion.content,
          status: assertion.status,
          priority: assertion.priority,
          created: assertion.created
        });
      }
    }
    
    return draftAssertions;
  }

  /**
   * Analyze dependencies using LLM reasoning
   * This is a simplified version - in production, this would call an LLM API
   * For now, we'll use a rule-based approach for basic dependency detection
   * 
   * @param {Array<Object>} assertions - Array of assertion objects
   * @returns {Object} Map of assertion-id -> { depends-on, reasoning }
   */
  analyzeDependencies(assertions) {
    const dependencies = {};
    
    // For each assertion, check if it references other assertions
    for (const assertion of assertions) {
      const content = assertion.content.toLowerCase();
      let dependsOn = null;
      let reasoning = 'No clear prerequisite identified';
      
      // Simple pattern matching to detect dependencies
      // Look for phrases like "depends on", "requires", "needs", etc.
      for (const other of assertions) {
        if (other.id === assertion.id) continue;
        
        const otherId = other.id.toLowerCase();
        const otherTitle = (other.title || '').toLowerCase();
        
        // Check if this assertion's content references the other assertion
        if (content.includes(otherId) || 
            (otherTitle && content.includes(otherTitle))) {
          // Found a potential dependency
          dependsOn = other.id;
          reasoning = `References ${other.id} in description`;
          break;
        }
        
        // Check for domain relationships
        // e.g., if assertion is about "message-input" and other is about "session-model"
        if (this.hasDomainDependency(assertion, other)) {
          dependsOn = other.id;
          reasoning = `Domain relationship: ${assertion.id} likely needs ${other.id}`;
          break;
        }
      }
      
      dependencies[assertion.id] = {
        'depends-on': dependsOn,
        reasoning: reasoning
      };
    }
    
    // Detect and reject circular dependencies
    this.detectCircularDependencies(dependencies);
    
    return dependencies;
  }

  /**
   * Simple heuristic for domain dependency detection
   * @param {Object} assertion
   * @param {Object} other
   * @returns {boolean}
   */
  hasDomainDependency(assertion, other) {
    // Example heuristics:
    // - Input/UI components depend on models/state
    // - Features depend on infrastructure
    
    const assertionId = assertion.id.toLowerCase();
    const otherId = other.id.toLowerCase();
    
    // UI/input components depend on models
    if (assertionId.includes('input') && otherId.includes('model')) {
      return true;
    }
    
    // Components depend on connections/infrastructure
    if (assertionId.includes('message') && otherId.includes('session')) {
      return true;
    }
    
    return false;
  }

  /**
   * Detect circular dependencies in the dependency map
   * @param {Object} dependencies - Map of assertion-id -> { depends-on, reasoning }
   * @throws {Error} If circular dependency detected
   */
  detectCircularDependencies(dependencies) {
    const visited = new Set();
    const recursionStack = new Set();
    
    const hasCycle = (assertionId) => {
      if (recursionStack.has(assertionId)) {
        return true; // Found a cycle
      }
      
      if (visited.has(assertionId)) {
        return false; // Already checked this path
      }
      
      visited.add(assertionId);
      recursionStack.add(assertionId);
      
      const dep = dependencies[assertionId];
      if (dep && dep['depends-on']) {
        if (hasCycle(dep['depends-on'])) {
          return true;
        }
      }
      
      recursionStack.delete(assertionId);
      return false;
    };
    
    for (const assertionId in dependencies) {
      if (hasCycle(assertionId)) {
        throw new Error(`Circular dependency detected involving: ${assertionId}`);
      }
    }
  }

  /**
   * Build a visual tree representation of dependencies
   * @param {Array<Object>} assertions
   * @param {Object} dependencies
   * @returns {string} Formatted tree
   */
  buildDependencyTree(assertions, dependencies) {
    let tree = 'Dependency Analysis\n';
    tree += '===================\n\n';
    
    // Group assertions by parent spec
    const byParent = {};
    for (const assertion of assertions) {
      if (!byParent[assertion.parent]) {
        byParent[assertion.parent] = [];
      }
      byParent[assertion.parent].push(assertion);
    }
    
    // Build tree for each parent spec
    for (const [parent, asserts] of Object.entries(byParent)) {
      tree += `${parent}:\n`;
      
      // Find root assertions (those with no dependencies in this group)
      const roots = asserts.filter(a => {
        const dep = dependencies[a.id];
        return !dep || !dep['depends-on'];
      });
      
      // Build tree from each root
      for (const root of roots) {
        tree += this.formatTreeNode(root, dependencies, asserts, '  ', new Set());
      }
      
      tree += '\n';
    }
    
    return tree;
  }

  /**
   * Format a single node in the dependency tree
   * @param {Object} assertion
   * @param {Object} dependencies
   * @param {Array<Object>} allAssertions
   * @param {string} prefix
   * @param {Set} visited
   * @returns {string}
   */
  formatTreeNode(assertion, dependencies, allAssertions, prefix, visited) {
    if (visited.has(assertion.id)) {
      return `${prefix}├─ ${assertion.id} (circular reference!)\n`;
    }
    
    visited.add(assertion.id);
    
    let output = `${prefix}├─ ${assertion.id}\n`;
    
    // Find children (assertions that depend on this one)
    const children = allAssertions.filter(a => {
      const dep = dependencies[a.id];
      return dep && dep['depends-on'] === assertion.id;
    });
    
    for (let i = 0; i < children.length; i++) {
      const child = children[i];
      const isLast = i === children.length - 1;
      const childPrefix = prefix + (isLast ? '  ' : '│ ');
      output += this.formatTreeNode(child, dependencies, allAssertions, childPrefix, visited);
    }
    
    return output;
  }

  /**
   * Update YAML frontmatter with depends-on field
   * @param {string} filePath - Path to assertion file
   * @param {string|null} dependsOn - ID of parent assertion or null
   */
  updateDependsOnField(filePath, dependsOn) {
    const content = fs.readFileSync(filePath, 'utf8');
    const lines = content.split('\n');
    
    // Find frontmatter boundaries
    let frontmatterStart = -1;
    let frontmatterEnd = -1;
    
    for (let i = 0; i < lines.length; i++) {
      if (lines[i] === '---') {
        if (frontmatterStart === -1) {
          frontmatterStart = i;
        } else {
          frontmatterEnd = i;
          break;
        }
      }
    }
    
    if (frontmatterStart === -1 || frontmatterEnd === -1) {
      throw new Error(`Invalid frontmatter in ${filePath}`);
    }
    
    // Check if depends-on field already exists
    let dependsOnLineIndex = -1;
    for (let i = frontmatterStart + 1; i < frontmatterEnd; i++) {
      if (lines[i].startsWith('depends-on:')) {
        dependsOnLineIndex = i;
        break;
      }
    }
    
    if (dependsOn) {
      // Add or update depends-on field
      const dependsOnLine = `depends-on: ${dependsOn}`;
      
      if (dependsOnLineIndex !== -1) {
        // Update existing field
        lines[dependsOnLineIndex] = dependsOnLine;
      } else {
        // Add new field after parent field
        let insertIndex = frontmatterEnd;
        for (let i = frontmatterStart + 1; i < frontmatterEnd; i++) {
          if (lines[i].startsWith('parent:')) {
            insertIndex = i + 1;
            break;
          }
        }
        lines.splice(insertIndex, 0, dependsOnLine);
      }
    } else {
      // Remove depends-on field if it exists
      if (dependsOnLineIndex !== -1) {
        lines.splice(dependsOnLineIndex, 1);
      }
    }
    
    fs.writeFileSync(filePath, lines.join('\n'), 'utf8');
  }

  /**
   * Update all assertion files with dependency information
   * @param {Object} dependencies - Map of assertion-id -> { depends-on, reasoning }
   * @param {Array<Object>} assertions - Array of assertion objects
   * @returns {Array<string>} Array of updated file paths
   */
  updateAssertionFiles(dependencies, assertions) {
    const updatedFiles = [];
    
    for (const assertion of assertions) {
      const dep = dependencies[assertion.id];
      if (dep) {
        this.updateDependsOnField(assertion.filePath, dep['depends-on']);
        updatedFiles.push(assertion.filePath);
      }
    }
    
    return updatedFiles;
  }

  /**
   * Commit dependency analysis changes
   * @param {Array<string>} filePaths - Array of file paths to commit
   * @param {string} baseDir - Working directory
   * @returns {Object} Commit result
   */
  commitDependencyUpdates(filePaths, baseDir = process.cwd()) {
    if (!filePaths || filePaths.length === 0) {
      throw new Error('No files to commit');
    }
    
    const execOpts = { cwd: baseDir, stdio: 'pipe', encoding: 'utf8' };
    
    // Stage files
    for (const filePath of filePaths) {
      const relativePath = path.relative(baseDir, filePath);
      execSync(`git add "${relativePath}"`, execOpts);
    }
    
    // Commit
    const message = `coordinator: Analyze and update assertion dependencies\n\nUpdated ${filePaths.length} assertion(s) with dependency metadata`;
    const msgFile = path.join(baseDir, '.git', 'COORDINATOR_COMMIT_MSG');
    fs.writeFileSync(msgFile, message, 'utf8');
    
    try {
      execSync(`git commit -F "${msgFile}"`, execOpts);
    } finally {
      if (fs.existsSync(msgFile)) {
        fs.unlinkSync(msgFile);
      }
    }
    
    const commitHash = execSync('git rev-parse --short HEAD', execOpts).trim();
    
    return { success: true, commitHash };
  }

  /**
   * Main orchestration method for dependency analysis
   * @param {string} baseDir - Project root directory
   * @returns {Object} Analysis result
   */
  async runDependencyAnalysis(baseDir = process.cwd()) {
    // 1. Read all draft/not_started assertions
    const assertions = this.readDraftAssertions(baseDir);
    
    if (assertions.length === 0) {
      return {
        success: true,
        message: 'No draft or not_started assertions found',
        assertionCount: 0
      };
    }
    
    // 2. Analyze dependencies
    const dependencies = this.analyzeDependencies(assertions);
    
    // 3. Build dependency tree
    const tree = this.buildDependencyTree(assertions, dependencies);
    
    return {
      success: true,
      assertionCount: assertions.length,
      dependencies: dependencies,
      tree: tree,
      assertions: assertions
    };
  }
}
