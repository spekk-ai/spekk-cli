import fs from 'node:fs';
import path from 'node:path';
import { execSync } from 'node:child_process';
import { Skill } from './skill-interface.js';

/**
 * Meeting Notes to Specs skill
 * Processes meeting transcripts and converts feature discussions into proper spec files.
 * This is a coach skill - it extends the coach's capabilities to handle meeting transcripts.
 */
export class MeetingNotesToSpecs extends Skill {
  constructor() {
    super();
    this.triggerKeywords = [
      'meeting notes', 'meeting transcript', 'meeting summary',
      'process meeting', 'from our meeting', 'discussed in meeting',
      'meeting action items', 'meeting outcomes', 'standup notes',
      'retro notes', 'planning notes', 'kickoff notes'
    ];
  }

  getId() {
    return 'meeting-notes-to-specs';
  }

  getName() {
    return 'Meeting Notes to Specs';
  }

  getDescription() {
    return 'Processes meeting transcripts and converts feature discussions into proper spec files with assertions, todos, and context updates.';
  }

  shouldTrigger(userInput, context = {}) {
    const lowerInput = userInput.toLowerCase();
    return this.triggerKeywords.some(keyword => lowerInput.includes(keyword));
  }

  getQuestions() {
    return [
      {
        id: 'transcript',
        text: 'Please provide the meeting transcript or notes:',
        type: 'text'
      }
    ];
  }

  processResponses(responses) {
    const transcript = responses.transcript || '';

    return {
      summary: 'Meeting transcript received. Features should be extracted and converted to specs.',
      recommendations: [
        'Extract feature discussions and convert to spec files',
        'Separate todos from features from decisions',
        'Propose spec structure before creating files'
      ],
      data: {
        transcript,
        outputTypes: ['specs', 'todos', 'context']
      }
    };
  }

  getOutputFormat() {
    return {
      filePattern: 'specs/{spec-id}/{spec-id}.md',
      format: 'markdown',
      sections: ['specs', 'todos', 'context']
    };
  }

  /**
   * Convert a feature description into a spec structure
   * @param {Object} feature - Feature object with id, title, description, priority, assertions
   * @returns {Object} Spec structure with parent and assertion files
   */
  featureToSpec(feature) {
    if (!feature.id || !feature.title) {
      throw new Error('Feature must have id and title');
    }

    const specId = this.toKebabCase(feature.id);
    const priority = this.validatePriority(feature.priority || 2);
    const created = feature.created || new Date().toISOString().replace(/\.\d{3}Z$/, 'Z');

    const parentSpec = this.generateParentSpec({
      id: specId,
      title: feature.title,
      description: feature.description || '',
      created,
      priority,
      successCriteria: feature.successCriteria || []
    });

    const assertions = (feature.assertions || []).map(assertion => {
      const assertionId = this.toKebabCase(assertion.id);
      return this.generateAssertion({
        id: assertionId,
        parent: specId,
        title: assertion.title,
        description: assertion.description || '',
        created,
        priority: this.validatePriority(assertion.priority || priority),
        successCriteria: assertion.successCriteria || []
      });
    });

    return {
      specId,
      parentSpec,
      assertions,
      directory: `specs/${specId}`,
      files: [
        { path: `specs/${specId}/${specId}.md`, content: parentSpec },
        ...assertions.map(a => ({
          path: `specs/${specId}/assertions/${a.id}.md`,
          content: a.content
        }))
      ]
    };
  }

  /**
   * Convert multiple features into multiple separate specs
   * @param {Array<Object>} features - Array of feature objects
   * @returns {Array<Object>} Array of spec structures
   */
  featuresToSpecs(features) {
    if (!Array.isArray(features) || features.length === 0) {
      throw new Error('Features must be a non-empty array');
    }

    return features.map(feature => this.featureToSpec(feature));
  }

  /**
   * Generate a proposal summary for user approval
   * @param {Array<Object>} specs - Array of spec structures from featuresToSpecs
   * @returns {string} Formatted proposal text
   */
  formatProposal(specs) {
    let proposal = 'Proposed Specs from Meeting\n\n';

    specs.forEach((spec, index) => {
      const parentLines = spec.parentSpec.split('\n');
      const titleLine = parentLines.find(l => l.startsWith('# ')) || `# ${spec.specId}`;
      const title = titleLine.replace('# ', '');

      // Extract priority from parent spec frontmatter
      const priorityMatch = spec.parentSpec.match(/priority:\s*(\d)/);
      const priority = priorityMatch ? priorityMatch[1] : '2';

      proposal += `Spec ${index + 1}: ${title}\n`;
      proposal += `  Priority: ${priority}\n`;
      proposal += `  Assertions:\n`;

      spec.assertions.forEach(assertion => {
        const assertionPriorityMatch = assertion.content.match(/priority:\s*(\d)/);
        const assertionPriority = assertionPriorityMatch ? assertionPriorityMatch[1] : '2';

        proposal += `    - ${assertion.title} (priority ${assertionPriority})\n`;

        if (assertion.successCriteria && assertion.successCriteria.length > 0) {
          proposal += `      Success: ${assertion.successCriteria[0]}\n`;
        }
      });

      proposal += '\n';
    });

    proposal += 'Shall I create these spec files?';
    return proposal;
  }

  /**
   * Generate parent spec markdown content
   * @param {Object} params - Spec parameters
   * @returns {string} Markdown content for parent spec
   */
  generateParentSpec({ id, title, description, created, priority, successCriteria }) {
    let content = '---\n';
    content += `id: ${id}\n`;
    content += `created: ${created}\n`;
    content += `priority: ${priority}\n`;
    content += '---\n\n';
    content += `# ${title}\n\n`;

    if (description) {
      content += `${description}\n\n`;
    }

    if (successCriteria && successCriteria.length > 0) {
      content += '## Success Criteria\n\n';
      successCriteria.forEach(criterion => {
        content += `- ${criterion}\n`;
      });
    }

    return content;
  }

  /**
   * Generate assertion markdown content
   * @param {Object} params - Assertion parameters
   * @returns {Object} Assertion object with id, title, content, and successCriteria
   */
  generateAssertion({ id, parent, title, description, created, priority, successCriteria }) {
    let content = '---\n';
    content += `id: ${id}\n`;
    content += `parent: ${parent}\n`;
    content += `created: ${created}\n`;
    content += `priority: ${priority}\n`;
    content += 'status: not_started\n';
    content += '---\n\n';
    content += `# ${title}\n\n`;

    if (description) {
      content += `${description}\n\n`;
    }

    if (successCriteria && successCriteria.length > 0) {
      content += '## Success Criteria\n\n';
      successCriteria.forEach(criterion => {
        content += `- ${criterion}\n`;
      });
    }

    return { id, title, content, successCriteria };
  }

  /**
   * Write spec files to disk
   * @param {Object} spec - Spec structure from featureToSpec
   * @param {string} baseDir - Base directory (defaults to cwd)
   * @returns {Array<string>} Array of created file paths
   */
  writeSpecFiles(spec, baseDir = process.cwd()) {
    const createdFiles = [];

    // Create spec directory
    const specDir = path.join(baseDir, spec.directory);
    const assertionsDir = path.join(specDir, 'assertions');

    if (!fs.existsSync(specDir)) {
      fs.mkdirSync(specDir, { recursive: true });
    }
    if (!fs.existsSync(assertionsDir)) {
      fs.mkdirSync(assertionsDir, { recursive: true });
    }

    // Write all files
    for (const file of spec.files) {
      const filePath = path.join(baseDir, file.path);
      fs.writeFileSync(filePath, file.content, 'utf8');
      createdFiles.push(file.path);
    }

    return createdFiles;
  }

  /**
   * Extract and validate three categories from meeting data.
   * Separates todos, features, and decisions into distinct outputs
   * for further processing by the coach.
   * @param {Object} meetingData - Categorized meeting data
   * @param {Array} meetingData.todos - Action items, follow-ups, assignments
   * @param {Array} meetingData.features - Product changes, new functionality
   * @param {Array} meetingData.decisions - Architectural decisions, patterns established
   * @returns {Object} Validated categories with summary
   */
  extractCategories(meetingData) {
    if (!meetingData || typeof meetingData !== 'object' || Array.isArray(meetingData)) {
      throw new Error('meetingData must be an object');
    }

    const { todos, features, decisions } = meetingData;

    // Validate category types when explicitly provided
    if (todos !== undefined && !Array.isArray(todos)) {
      throw new Error('todos must be an array');
    }
    if (features !== undefined && !Array.isArray(features)) {
      throw new Error('features must be an array');
    }
    if (decisions !== undefined && !Array.isArray(decisions)) {
      throw new Error('decisions must be an array');
    }

    const validTodos = todos || [];
    const validFeatures = features || [];
    const validDecisions = decisions || [];

    return {
      todos: validTodos,
      features: validFeatures,
      decisions: validDecisions,
      summary: {
        todoCount: validTodos.length,
        featureCount: validFeatures.length,
        decisionCount: validDecisions.length,
        totalItems: validTodos.length + validFeatures.length + validDecisions.length
      }
    };
  }

  /**
   * Convert a string to kebab-case
   * @param {string} str - Input string
   * @returns {string} Kebab-case string
   */
  toKebabCase(str) {
    return str
      .replace(/([a-z])([A-Z])/g, '$1-$2')
      .replace(/[\s_]+/g, '-')
      .replace(/[^a-zA-Z0-9-]/g, '')
      .toLowerCase()
      .replace(/-+/g, '-')
      .replace(/^-|-$/g, '');
  }

  /**
   * Validate and coerce priority to valid range (1-3)
   * @param {number} priority - Priority value
   * @returns {number} Valid priority (1, 2, or 3)
   */
  validatePriority(priority) {
    const num = parseInt(priority, 10);
    if (isNaN(num) || num < 1) return 1;
    if (num > 3) return 3;
    return num;
  }

  /**
   * Format decisions as markdown list items with date stamps
   * @param {Array<{decision: string, context?: string}>} decisions
   * @param {string} meetingDate - Date string like "2025-02-12"
   * @returns {string} Formatted markdown
   */
  formatDecisions(decisions, meetingDate) {
    if (!decisions || decisions.length === 0) return '';

    return decisions.map(d => {
      let entry = `- Decision from meeting ${meetingDate}: ${d.decision}`;
      if (d.context) {
        entry += `\n  - *Context: ${d.context}*`;
      }
      return entry;
    }).join('\n') + '\n';
  }

  /**
   * Generate updated CONTEXT.md content
   * @param {Array<{decision: string, context?: string}>} decisions
   * @param {string} meetingDate
   * @param {string|null} existingContent - Current CONTEXT.md content or null
   * @returns {string} Updated content (empty string if no decisions)
   */
  generateContextUpdate(decisions, meetingDate, existingContent) {
    if (!decisions || decisions.length === 0) return '';

    const formatted = this.formatDecisions(decisions, meetingDate);

    if (!existingContent) {
      return `# Project Context\n\n## Architectural Decisions\n\n${formatted}`;
    }

    // If existing content has an Architectural Decisions section, append there
    const sectionHeader = '## Architectural Decisions';
    const sectionIndex = existingContent.indexOf(sectionHeader);

    if (sectionIndex === -1) {
      // No section yet — append at end
      return existingContent.trimEnd() + `\n\n${sectionHeader}\n\n${formatted}`;
    }

    // Find the end of the Architectural Decisions section (next ## heading or EOF)
    const afterHeader = sectionIndex + sectionHeader.length;
    const nextSectionMatch = existingContent.slice(afterHeader).search(/\n## /);

    if (nextSectionMatch === -1) {
      // No next section — append at end
      return existingContent.trimEnd() + '\n' + formatted;
    }

    // Insert before next section
    const insertPoint = afterHeader + nextSectionMatch;
    const before = existingContent.slice(0, insertPoint).trimEnd();
    const after = existingContent.slice(insertPoint);
    return before + '\n' + formatted + after;
  }

  /**
   * Generate a simple diff showing changes to CONTEXT.md
   * @param {string|null} oldContent
   * @param {string} newContent
   * @returns {string} Human-readable diff
   */
  generateContextDiff(oldContent, newContent) {
    if (!oldContent) {
      const lines = newContent.split('\n').map(l => `+ ${l}`).join('\n');
      return `CONTEXT.md (new file)\n${lines}`;
    }

    const oldLines = oldContent.split('\n');
    const newLines = newContent.split('\n');
    const oldSet = new Set(oldLines);
    const additions = newLines.filter(l => !oldSet.has(l));

    return `CONTEXT.md (updated)\n${additions.map(l => `+ ${l}`).join('\n')}`;
  }

  /**
   * Read CONTEXT.md from a directory
   * @param {string} baseDir
   * @returns {string|null} File content or null if not found
   */
  readContextFile(baseDir = process.cwd()) {
    const filePath = path.join(baseDir, 'CONTEXT.md');
    if (!fs.existsSync(filePath)) return null;
    return fs.readFileSync(filePath, 'utf8');
  }

  /**
   * Write CONTEXT.md to a directory
   * @param {string} content
   * @param {string} baseDir
   */
  writeContextFile(content, baseDir = process.cwd()) {
    fs.writeFileSync(path.join(baseDir, 'CONTEXT.md'), content, 'utf8');
  }

  /**
   * Build a commit message for meeting outputs.
   * Format: "Process meeting: {date} - {summary}" with categorized body.
   * @param {Object} params
   * @param {string} params.meetingDate - Date string like "2025-02-12"
   * @param {string} params.summary - Brief summary of meeting outcomes
   * @param {Array<string>} params.todos - Todo descriptions
   * @param {Array<string>} params.specIds - IDs of specs created
   * @param {Array<string>} params.decisions - Decision descriptions
   * @returns {string} Formatted commit message
   */
  buildCommitMessage({ meetingDate, summary, todos = [], specIds = [], decisions = [] }) {
    if (!meetingDate) throw new Error('meetingDate is required');
    if (!summary) throw new Error('summary is required');

    let message = `Process meeting: ${meetingDate} - ${summary}`;

    const sections = [];

    if (todos.length > 0) {
      sections.push('Todos:\n' + todos.map(t => `- ${t}`).join('\n'));
    }

    if (specIds.length > 0) {
      sections.push('Specs created:\n' + specIds.map(s => `- ${s}`).join('\n'));
    }

    if (decisions.length > 0) {
      sections.push('Context updates:\n' + decisions.map(d => `- ${d}`).join('\n'));
    }

    if (sections.length > 0) {
      message += '\n\n' + sections.join('\n\n');
    }

    return message;
  }

  /**
   * Stage and commit all meeting outputs in a single git commit.
   * @param {Object} params
   * @param {string} params.meetingDate - Date string like "2025-02-12"
   * @param {string} params.summary - Brief summary of meeting outcomes
   * @param {Array<string>} params.todos - Todo descriptions
   * @param {Array<string>} params.specIds - IDs of specs created
   * @param {Array<string>} params.decisions - Decision descriptions
   * @param {Array<string>} params.filePaths - Relative file paths to stage
   * @param {string} [params.baseDir] - Working directory (defaults to cwd)
   * @returns {{ success: boolean, commitHash: string }} Result
   */
  commitAllOutputs({ meetingDate, summary, todos = [], specIds = [], decisions = [], filePaths, baseDir = process.cwd() }) {
    if (!filePaths || filePaths.length === 0) {
      throw new Error('filePaths must be a non-empty array');
    }

    const commitMessage = this.buildCommitMessage({ meetingDate, summary, todos, specIds, decisions });
    const execOpts = { cwd: baseDir, stdio: 'pipe', encoding: 'utf8' };

    // Stage specific files
    for (const filePath of filePaths) {
      execSync(`git add "${filePath}"`, execOpts);
    }

    // Write commit message to a temp file to avoid shell escaping issues
    const msgFile = path.join(baseDir, '.git', 'MEETING_COMMIT_MSG');
    fs.writeFileSync(msgFile, commitMessage, 'utf8');

    try {
      execSync(`git commit -F "${msgFile}"`, execOpts);
    } finally {
      // Clean up temp message file
      if (fs.existsSync(msgFile)) {
        fs.unlinkSync(msgFile);
      }
    }

    // Get the commit hash
    const commitHash = execSync('git rev-parse --short HEAD', execOpts).trim();

    return { success: true, commitHash };
  }
}
