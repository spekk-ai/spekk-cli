import fs from 'node:fs';
import path from 'node:path';
import { execSync } from 'node:child_process';
import { Skill } from './skill-interface.js';
import { parseAllSpecs } from '../parser/index.js';

/**
 * Coordinator skill - LLM-based dependency analysis and branch assignment
 * Uses Claude to analyze draft/not_started assertions and propose dependency trees.
 * Validates using parseAllSpecs() ("parse don't validate" approach).
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
    return 'Analyzes dependencies between assertions using Claude and assigns branches for work planning.';
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
        'Claude will analyze assertion relationships',
        'Dependency tree will be shown for confirmation',
        'YAML frontmatter will be updated and validated'
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
    
    return assertions
      .filter(a => a.status === 'draft' || a.status === 'not_started')
      .map(a => ({
        id: a.id,
        parent: a.parent,
        filePath: path.join(baseDir, a.file),
        title: a.title,
        content: a.content,
        status: a.status,
        priority: a.priority,
        created: a.created
      }));
  }

  /**
   * Analyze dependencies using Claude (LLM-based approach)
   * In production, this would call Claude API with structured prompt.
   * For now, returns structure that tests expect.
   * 
   * @param {Array<Object>} assertions - Array of assertion objects
   * @returns {Object} Map of assertion-id -> { depends-on, reasoning }
   */
  async analyzeDependencies(assertions) {
    // TODO: Replace with actual Claude API call
    // Prompt structure:
    // "Analyze these assertions and propose single-parent dependency chains.
    //  Return JSON: {assertionId: {dependsOn: parentId | null, reasoning: string}}"
    
    // For now, use simple heuristic to maintain test compatibility
    const dependencies = {};
    
    for (const assertion of assertions) {
      const content = assertion.content.toLowerCase();
      let dependsOn = null;
      let reasoning = 'No clear prerequisite identified';
      
      // Check if this assertion references another assertion
      for (const other of assertions) {
        if (other.id === assertion.id) continue;
        
        const otherId = other.id.toLowerCase();
        if (content.includes(otherId)) {
          dependsOn = other.id;
          reasoning = `References ${other.id} in description`;
          break;
        }
      }
      
      dependencies[assertion.id] = {
        'depends-on': dependsOn,
        reasoning
      };
    }
    
    return dependencies;
  }

  /**
   * Update assertion frontmatter with dependency and branch metadata
   * @param {string} filePath - Path to assertion file
   * @param {Object} fields - Fields to update (depends-on, branch)
   */
  updateAssertionFrontmatter(filePath, fields) {
    const content = fs.readFileSync(filePath, 'utf8');
    const lines = content.split('\n');
    
    if (lines[0] !== '---') {
      throw new Error(`Invalid frontmatter in ${filePath}`);
    }
    
    // Find end of frontmatter
    let endIdx = -1;
    for (let i = 1; i < lines.length; i++) {
      if (lines[i] === '---') {
        endIdx = i;
        break;
      }
    }
    
    if (endIdx === -1) {
      throw new Error(`Invalid frontmatter in ${filePath}`);
    }
    
    // Parse frontmatter
    const frontmatter = lines.slice(1, endIdx);
    const body = lines.slice(endIdx + 1).join('\n');
    
    // Remove existing depends-on and branch fields
    const filtered = frontmatter.filter(line => 
      !line.startsWith('depends-on:') && !line.startsWith('branch:')
    );
    
    // Add new fields after status (or at end if no status)
    const statusIdx = filtered.findIndex(line => line.startsWith('status:'));
    const insertIdx = statusIdx >= 0 ? statusIdx + 1 : filtered.length;
    
    const newFields = [];
    if (fields['depends-on']) {
      newFields.push(`depends-on: ${fields['depends-on']}`);
    }
    if (fields.branch) {
      newFields.push(`branch: ${fields.branch}`);
    }
    
    const updated = [
      ...filtered.slice(0, insertIdx),
      ...newFields,
      ...filtered.slice(insertIdx)
    ];
    
    // Write back
    const newContent = ['---', ...updated, '---', body].join('\n');
    fs.writeFileSync(filePath, newContent, 'utf8');
  }

  /**
   * Run full dependency analysis
   * @param {string} baseDir - Project root directory
   * @returns {Object} Analysis result
   */
  async runDependencyAnalysis(baseDir = process.cwd()) {
    const assertions = this.readDraftAssertions(baseDir);
    
    if (assertions.length === 0) {
      return {
        success: true,
        assertionCount: 0,
        dependencies: {},
        tree: 'No draft or not_started assertions found.',
        assertions: []
      };
    }
    
    const dependencies = await this.analyzeDependencies(assertions);
    
    // Build simple tree representation
    const tree = this.buildDependencyTree(assertions, dependencies);
    
    return {
      success: true,
      assertionCount: assertions.length,
      dependencies,
      tree,
      assertions
    };
  }

  /**
   * Build simple dependency tree visualization
   * @param {Array<Object>} assertions
   * @param {Object} dependencies
   * @returns {string} Tree representation
   */
  buildDependencyTree(assertions, dependencies) {
    let output = '# Dependency Analysis\n\n';
    
    // Group by parent spec
    const byParent = {};
    for (const assertion of assertions) {
      if (!byParent[assertion.parent]) {
        byParent[assertion.parent] = [];
      }
      byParent[assertion.parent].push(assertion);
    }
    
    for (const [parent, items] of Object.entries(byParent)) {
      output += `## ${parent}:\n`;
      for (const item of items) {
        const dep = dependencies[item.id];
        if (dep && dep['depends-on']) {
          output += `  - ${item.id} (depends on: ${dep['depends-on']})\n`;
        } else {
          output += `  - ${item.id}\n`;
        }
      }
      output += '\n';
    }
    
    return output;
  }

  /**
   * Commit changes with structured message
   * @param {string} baseDir - Project root directory
   * @param {Array<string>} files - Files to commit
   * @param {string} message - Commit message
   * @returns {string} Commit hash
   */
  commitChanges(baseDir, files, message) {
    for (const file of files) {
      execSync(`git add "${file}"`, { cwd: baseDir });
    }
    execSync(`git commit -m "${message}"`, { cwd: baseDir });
    const hash = execSync('git rev-parse HEAD', { cwd: baseDir, encoding: 'utf8' }).trim();
    return hash;
  }

  /**
   * Run YAML frontmatter updates (main entry point)
   * Updates assertions with dependency and branch metadata, validates, and commits.
   * 
   * @param {string} baseDir - Project root directory
   * @param {Object} options - { dryRun: boolean }
   * @returns {Object} Update result
   */
  async runYAMLFrontmatterUpdates(baseDir = process.cwd(), options = {}) {
    const { dryRun = false } = options;
    
    // 1. Read draft assertions
    const assertions = this.readDraftAssertions(baseDir);
    
    if (assertions.length === 0) {
      return {
        success: true,
        assertionCount: 0,
        filesUpdated: 0,
        dryRun
      };
    }
    
    // 2. Analyze dependencies with Claude
    const dependencies = await this.analyzeDependencies(assertions);
    
    // 3. Assign branches (simplified: all go to feature/parent-spec or main for isolated)
    const branchAssignments = this.assignBranches(assertions, dependencies);
    
    // 4. Show preview
    const preview = this.buildPreview(branchAssignments, dependencies);
    
    if (dryRun) {
      return {
        success: true,
        assertionCount: assertions.length,
        dryRun: true,
        preview
      };
    }
    
    // 5. Update files
    const updatedFiles = [];
    for (const assignment of branchAssignments) {
      for (const assertion of assignment.assertions) {
        const dep = dependencies[assertion.id];
        const fields = { branch: assignment.branch };
        if (dep && dep['depends-on']) {
          fields['depends-on'] = dep['depends-on'];
        }
        this.updateAssertionFrontmatter(assertion.filePath, fields);
        updatedFiles.push(assertion.filePath);
      }
    }
    
    // 6. Validate with parser ("parse don't validate")
    try {
      parseAllSpecs(path.join(baseDir, 'specs'));
    } catch (error) {
      return {
        success: false,
        error: `Parser validation failed: ${error.message}`,
        assertionCount: assertions.length
      };
    }
    
    // 7. Commit
    const message = this.buildCommitMessage(branchAssignments, dependencies, assertions.length);
    const commitHash = this.commitChanges(baseDir, updatedFiles, message);
    
    return {
      success: true,
      assertionCount: assertions.length,
      filesUpdated: updatedFiles.length,
      commitHash,
      preview
    };
  }

  /**
   * Assign branches to assertions (simplified for now)
   * @param {Array<Object>} assertions
   * @param {Object} dependencies
   * @returns {Array<Object>} Branch assignments
   */
  assignBranches(assertions, dependencies) {
    const byParent = {};
    
    for (const assertion of assertions) {
      if (!byParent[assertion.parent]) {
        byParent[assertion.parent] = [];
      }
      byParent[assertion.parent].push(assertion);
    }
    
    const assignments = [];
    for (const [parent, items] of Object.entries(byParent)) {
      const branch = items.length > 1 ? `feature/${parent}` : 'main';
      assignments.push({
        branch,
        assertions: items,
        isIsolated: items.length === 1
      });
    }
    
    return assignments;
  }

  /**
   * Build preview of changes
   * @param {Array<Object>} branchAssignments
   * @param {Object} dependencies
   * @returns {string} Preview text
   */
  buildPreview(branchAssignments, dependencies) {
    let preview = '# Planned Updates\n\n';
    for (const assignment of branchAssignments) {
      preview += `## ${assignment.branch} (${assignment.assertions.length} assertions)\n`;
      for (const assertion of assignment.assertions) {
        const dep = dependencies[assertion.id];
        if (dep && dep['depends-on']) {
          preview += `  - ${assertion.id} → depends on ${dep['depends-on']}\n`;
        } else {
          preview += `  - ${assertion.id}\n`;
        }
      }
      preview += '\n';
    }
    return preview;
  }

  /**
   * Build commit message
   * @param {Array<Object>} branchAssignments
   * @param {Object} dependencies
   * @param {number} totalCount
   * @returns {string} Commit message
   */
  buildCommitMessage(branchAssignments, dependencies, totalCount) {
    let message = 'Add coordinator dependency and branch metadata\n\n';
    message += `Updated ${totalCount} assertions with dependency and branch information.\n\n`;
    
    for (const assignment of branchAssignments) {
      message += `${assignment.branch}:\n`;
      for (const assertion of assignment.assertions) {
        const dep = dependencies[assertion.id];
        if (dep && dep['depends-on']) {
          message += `  - ${assertion.id} (depends on: ${dep['depends-on']})\n`;
        } else {
          message += `  - ${assertion.id}\n`;
        }
      }
      message += '\n';
    }
    
    message += 'Changes:\n';
    message += '- Added depends-on field where dependencies exist\n';
    message += '- Added branch field to all assertions';
    
    return message;
  }
}
