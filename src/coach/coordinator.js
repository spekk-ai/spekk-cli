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
   * Update YAML frontmatter with depends-on and branch fields
   * @param {string} filePath - Path to assertion file
   * @param {Object} updates - { depends-on: string|null, branch: string }
   */
  updateAssertionFrontmatter(filePath, updates) {
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
    
    // Find existing field indices
    let dependsOnLineIndex = -1;
    let branchLineIndex = -1;
    let statusLineIndex = -1;
    
    for (let i = frontmatterStart + 1; i < frontmatterEnd; i++) {
      if (lines[i].startsWith('depends-on:')) {
        dependsOnLineIndex = i;
      } else if (lines[i].startsWith('branch:')) {
        branchLineIndex = i;
      } else if (lines[i].startsWith('status:')) {
        statusLineIndex = i;
      }
    }
    
    // Determine insertion point (after status field)
    let insertionPoint = frontmatterEnd;
    if (statusLineIndex !== -1) {
      insertionPoint = statusLineIndex + 1;
    }
    
    // Track how many lines we've added/removed
    let linesAdded = 0;
    
    // Update depends-on field (only if provided in updates)
    if ('depends-on' in updates) {
      const dependsOn = updates['depends-on'];
      
      if (dependsOn) {
        // Add or update depends-on field
        const dependsOnLine = `depends-on: ${dependsOn}`;
        
        if (dependsOnLineIndex !== -1) {
          // Update existing field
          lines[dependsOnLineIndex] = dependsOnLine;
        } else {
          // Insert new field at insertion point
          lines.splice(insertionPoint, 0, dependsOnLine);
          linesAdded++;
          frontmatterEnd++;
          
          // Update branchLineIndex if it's after the insertion
          if (branchLineIndex !== -1 && branchLineIndex >= insertionPoint) {
            branchLineIndex++;
          }
        }
      } else {
        // Remove depends-on field if it exists
        if (dependsOnLineIndex !== -1) {
          lines.splice(dependsOnLineIndex, 1);
          linesAdded--;
          frontmatterEnd--;
          
          // Update indices that are after the removed line
          if (branchLineIndex !== -1 && branchLineIndex > dependsOnLineIndex) {
            branchLineIndex--;
          }
          if (insertionPoint > dependsOnLineIndex) {
            insertionPoint--;
          }
        }
      }
    }
    
    // Update branch field
    if ('branch' in updates) {
      const branchLine = `branch: ${updates.branch}`;
      
      if (branchLineIndex !== -1) {
        // Update existing field
        lines[branchLineIndex] = branchLine;
      } else {
        // Insert new field after depends-on (if exists) or at insertion point
        let branchInsertPoint = insertionPoint;
        
        // If we just added depends-on, insert branch right after it
        if ('depends-on' in updates && updates['depends-on']) {
          if (dependsOnLineIndex !== -1) {
            branchInsertPoint = dependsOnLineIndex + 1;
          } else {
            branchInsertPoint = insertionPoint + linesAdded;
          }
        }
        
        lines.splice(branchInsertPoint, 0, branchLine);
        frontmatterEnd++;
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
   * Update all assertion files with dependency and branch metadata
   * @param {Array<Object>} branchAssignments - Array of { branch, assertions }
   * @param {Object} dependencies - Map of assertion-id -> { depends-on, reasoning }
   * @param {string} baseDir - Project root directory
   * @returns {Array<string>} Array of updated file paths
   */
  updateAssertionFilesWithBranchMetadata(branchAssignments, dependencies, baseDir = process.cwd()) {
    const updatedFiles = [];
    
    for (const assignment of branchAssignments) {
      for (const assertion of assignment.assertions) {
        const updates = {
          branch: assignment.branch
        };
        
        // Add depends-on field only if dependency exists
        const dep = dependencies[assertion.id];
        if (dep && dep['depends-on']) {
          updates['depends-on'] = dep['depends-on'];
        }
        
        this.updateAssertionFrontmatter(assertion.filePath, updates);
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
   * Format dependency tree for a branch assignment
   * @param {Array<Object>} assertions - Array of assertions in this branch
   * @param {Object} dependencies - Map of assertion-id -> { depends-on, reasoning }
   * @returns {string} Formatted tree representation
   */
  formatBranchDependencyTree(assertions, dependencies) {
    // Build adjacency lists
    const dependents = {};
    const roots = [];
    
    for (const assertion of assertions) {
      dependents[assertion.id] = [];
    }
    
    for (const assertion of assertions) {
      const dep = dependencies[assertion.id];
      if (dep && dep['depends-on']) {
        if (dependents[dep['depends-on']]) {
          dependents[dep['depends-on']].push(assertion.id);
        }
      } else {
        roots.push(assertion.id);
      }
    }
    
    // Build tree string recursively
    const visited = new Set();
    
    const buildTree = (assertionId, prefix = '  - ') => {
      if (visited.has(assertionId)) {
        return '';
      }
      visited.add(assertionId);
      
      let tree = `${prefix}${assertionId}`;
      const children = dependents[assertionId] || [];
      
      if (children.length > 0) {
        tree += ' → ' + children.join(' → ');
      }
      
      return tree + '\n';
    };
    
    let output = '';
    for (const root of roots) {
      output += buildTree(root);
    }
    
    return output;
  }

  /**
   * Build comprehensive commit message for YAML frontmatter updates
   * @param {Array<Object>} branchAssignments - Array of { branch, assertions }
   * @param {Object} dependencies - Map of assertion-id -> { depends-on, reasoning }
   * @param {number} fileCount - Number of files updated
   * @returns {string} Commit message
   */
  buildCommitMessage(branchAssignments, dependencies, fileCount) {
    let message = 'Add coordinator dependency and branch metadata\n\n';
    message += 'Applied coordinator skill to organize work:\n\n';
    
    // Sort assignments: feature branches first, then main
    const sorted = branchAssignments.sort((a, b) => {
      if (a.branch === 'main') return 1;
      if (b.branch === 'main') return -1;
      return a.branch.localeCompare(b.branch);
    });
    
    for (const assignment of sorted) {
      const count = assignment.assertions.length;
      const label = assignment.branch === 'main' ?
        `${assignment.branch} (${count} assertion${count !== 1 ? 's' : ''})` :
        `${assignment.branch} (${count} assertion${count !== 1 ? 's' : ''})`;
      
      message += `${label}:\n`;
      message += this.formatBranchDependencyTree(assignment.assertions, dependencies);
      message += '\n';
    }
    
    message += 'Changes:\n';
    message += '- Added depends-on field where dependencies exist\n';
    message += '- Added branch field to all assertions\n';
    message += '- No changes to spec content or existing metadata';
    
    return message;
  }

  /**
   * Commit YAML frontmatter updates with comprehensive message
   * @param {Array<string>} filePaths - Array of file paths to commit
   * @param {Array<Object>} branchAssignments - Array of { branch, assertions }
   * @param {Object} dependencies - Map of assertion-id -> { depends-on, reasoning }
   * @param {string} baseDir - Working directory
   * @returns {Object} Commit result
   */
  commitYAMLFrontmatterUpdates(filePaths, branchAssignments, dependencies, baseDir = process.cwd()) {
    if (!filePaths || filePaths.length === 0) {
      throw new Error('No files to commit');
    }
    
    const execOpts = { cwd: baseDir, stdio: 'pipe', encoding: 'utf8' };
    
    // Stage files
    for (const filePath of filePaths) {
      const relativePath = path.relative(baseDir, filePath);
      execSync(`git add "${relativePath}"`, execOpts);
    }
    
    // Build comprehensive commit message
    const message = this.buildCommitMessage(branchAssignments, dependencies, filePaths.length);
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

  /**
   * Identify connected components (clusters) in the dependency graph
   * Assertions that share dependencies or are in a dependency chain belong to the same cluster
   * @param {Array<Object>} assertions - Array of assertion objects
   * @param {Object} dependencies - Map of assertion-id -> { depends-on, reasoning }
   * @returns {Array<Array<Object>>} Array of clusters (each cluster is an array of assertions)
   */
  identifyDependencyClusters(assertions, dependencies) {
    const visited = new Set();
    const clusters = [];
    
    // Build adjacency lists for both directions (depends-on and dependents)
    const dependents = {}; // Map of assertion-id -> assertions that depend on it
    for (const assertion of assertions) {
      dependents[assertion.id] = [];
    }
    for (const assertion of assertions) {
      const dep = dependencies[assertion.id];
      if (dep && dep['depends-on']) {
        if (dependents[dep['depends-on']]) {
          dependents[dep['depends-on']].push(assertion);
        }
      }
    }
    
    // DFS to find connected component
    const dfs = (assertionId, cluster) => {
      if (visited.has(assertionId)) return;
      visited.add(assertionId);
      
      const assertion = assertions.find(a => a.id === assertionId);
      if (!assertion) return;
      
      cluster.push(assertion);
      
      // Visit dependency (parent)
      const dep = dependencies[assertion.id];
      if (dep && dep['depends-on']) {
        dfs(dep['depends-on'], cluster);
      }
      
      // Visit dependents (children)
      if (dependents[assertionId]) {
        for (const dependent of dependents[assertionId]) {
          dfs(dependent.id, cluster);
        }
      }
    };
    
    // Find all connected components
    for (const assertion of assertions) {
      if (!visited.has(assertion.id)) {
        const cluster = [];
        dfs(assertion.id, cluster);
        clusters.push(cluster);
      }
    }
    
    return clusters;
  }

  /**
   * Generate a semantic branch name for a cluster of assertions
   * @param {Array<Object>} cluster - Array of assertion objects
   * @returns {string} Branch name in format feature/<name>
   */
  generateBranchName(cluster) {
    // Try parent spec ID first (if all assertions share same parent)
    const parentSpecs = [...new Set(cluster.map(a => a.parent))];
    
    if (parentSpecs.length === 1) {
      return `feature/${parentSpecs[0]}`;
    }
    
    // Multiple parent specs - use the most common one
    const parentCounts = {};
    for (const assertion of cluster) {
      parentCounts[assertion.parent] = (parentCounts[assertion.parent] || 0) + 1;
    }
    
    const mostCommon = Object.entries(parentCounts)
      .sort((a, b) => b[1] - a[1])[0][0];
    
    return `feature/${mostCommon}`;
  }

  /**
   * Assign branch names to clusters
   * @param {Array<Array<Object>>} clusters - Array of clusters
   * @returns {Array<Object>} Array of { branch, assertions, isIsolated }
   */
  assignBranchesToClusters(clusters) {
    const branchAssignments = [];
    
    for (const cluster of clusters) {
      const isIsolated = cluster.length === 1;
      const branchName = isIsolated ? 'main' : this.generateBranchName(cluster);
      
      branchAssignments.push({
        branch: branchName,
        assertions: cluster,
        isIsolated: isIsolated
      });
    }
    
    // Merge all isolated assertions into a single "main" group
    const isolated = branchAssignments.filter(b => b.isIsolated);
    const nonIsolated = branchAssignments.filter(b => !b.isIsolated);
    
    if (isolated.length > 0) {
      const mainAssertions = isolated.flatMap(b => b.assertions);
      nonIsolated.push({
        branch: 'main',
        assertions: mainAssertions,
        isIsolated: false
      });
    }
    
    return nonIsolated;
  }

  /**
   * Format branch assignments for display
   * @param {Array<Object>} branchAssignments - Array of { branch, assertions, isIsolated }
   * @returns {string} Formatted output
   */
  formatBranchAssignments(branchAssignments) {
    let output = 'Branch Assignments\n';
    output += '==================\n\n';
    
    // Sort by branch name (feature branches first, then main)
    const sorted = branchAssignments.sort((a, b) => {
      if (a.branch === 'main') return 1;
      if (b.branch === 'main') return -1;
      return a.branch.localeCompare(b.branch);
    });
    
    for (const assignment of sorted) {
      const count = assignment.assertions.length;
      const label = assignment.branch === 'main' ? 
        `${assignment.branch} (${count} isolated assertion${count !== 1 ? 's' : ''})` :
        `${assignment.branch} (${count} assertion${count !== 1 ? 's' : ''})`;
      
      output += `${label}:\n`;
      
      for (const assertion of assignment.assertions) {
        output += `  - ${assertion.id}\n`;
      }
      
      output += '\n';
    }
    
    return output;
  }

  /**
   * Check if a branch name conflicts with existing git branches
   * @param {string} branchName - Branch name to check
   * @param {string} baseDir - Project root directory
   * @returns {boolean} True if branch exists
   */
  branchExists(branchName, baseDir = process.cwd()) {
    const execOpts = { cwd: baseDir, stdio: 'pipe', encoding: 'utf8' };
    
    try {
      const branches = execSync('git branch -a', execOpts).toString();
      const branchPattern = new RegExp(`\\b${branchName}\\b`);
      return branchPattern.test(branches);
    } catch (error) {
      return false;
    }
  }

  /**
   * Main orchestration method for branch assignment
   * @param {string} baseDir - Project root directory
   * @returns {Object} Branch assignment result
   */
  async runBranchAssignment(baseDir = process.cwd()) {
    // 1. Read all draft/not_started assertions
    const assertions = this.readDraftAssertions(baseDir);
    
    if (assertions.length === 0) {
      return {
        success: true,
        message: 'No draft or not_started assertions found',
        assertionCount: 0
      };
    }
    
    // 2. Analyze dependencies (if not already done)
    const dependencies = this.analyzeDependencies(assertions);
    
    // 3. Identify clusters
    const clusters = this.identifyDependencyClusters(assertions, dependencies);
    
    // 4. Assign branches to clusters
    const branchAssignments = this.assignBranchesToClusters(clusters);
    
    // 5. Check for existing branches and warn
    const warnings = [];
    for (const assignment of branchAssignments) {
      if (assignment.branch !== 'main' && this.branchExists(assignment.branch, baseDir)) {
        warnings.push(`Branch '${assignment.branch}' already exists`);
      }
    }
    
    // 6. Format output
    const output = this.formatBranchAssignments(branchAssignments);
    
    return {
      success: true,
      assertionCount: assertions.length,
      clusterCount: clusters.length,
      branchAssignments: branchAssignments,
      dependencies: dependencies,
      output: output,
      warnings: warnings
    };
  }

  /**
   * Main orchestration method for YAML frontmatter updates
   * Analyzes dependencies, assigns branches, and updates all assertion files
   * @param {string} baseDir - Project root directory
   * @param {Object} options - Options { dryRun: boolean }
   * @returns {Object} Update result
   */
  async runYAMLFrontmatterUpdates(baseDir = process.cwd(), options = {}) {
    // 1. Run branch assignment to get dependencies and branch assignments
    const branchResult = await this.runBranchAssignment(baseDir);
    
    if (branchResult.assertionCount === 0) {
      return {
        success: true,
        message: 'No draft or not_started assertions found',
        assertionCount: 0
      };
    }
    
    const { branchAssignments, dependencies, warnings } = branchResult;
    
    // 2. Preview changes
    const preview = this.buildCommitMessage(branchAssignments, dependencies, branchResult.assertionCount);
    
    if (options.dryRun) {
      return {
        success: true,
        dryRun: true,
        assertionCount: branchResult.assertionCount,
        preview: preview,
        warnings: warnings
      };
    }
    
    // 3. Update all assertion files
    const updatedFiles = this.updateAssertionFilesWithBranchMetadata(
      branchAssignments,
      dependencies,
      baseDir
    );
    
    // 4. Commit changes
    const commitResult = this.commitYAMLFrontmatterUpdates(
      updatedFiles,
      branchAssignments,
      dependencies,
      baseDir
    );
    
    return {
      success: true,
      assertionCount: branchResult.assertionCount,
      filesUpdated: updatedFiles.length,
      commitHash: commitResult.commitHash,
      branchAssignments: branchAssignments,
      warnings: warnings
    };
  }
}
