import { readFileSync, readdirSync, existsSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { Skill } from './skill-interface.js';

/**
 * Loads skills from markdown files
 * Scans specs/coach-skills-system/ for skill markdown files
 */
export class MarkdownSkillLoader {
  /**
   * Scan directory for skill markdown files
   * @param {string} baseDir - Base directory containing specs/
   * @returns {Array<string>} Array of skill file paths
   */
  static scanForSkills(baseDir = process.cwd()) {
    const skillsDir = join(baseDir, 'specs', 'coach-skills-system');
    
    if (!existsSync(skillsDir)) {
      return [];
    }

    const files = [];
    const entries = readdirSync(skillsDir, { withFileTypes: true });

    for (const entry of entries) {
      const fullPath = join(skillsDir, entry.name);
      
      // Check for files ending in -skill.md (but not in assertions/ directory)
      // assertions/ contains spec assertions, not executable skills
      if (entry.isFile() && entry.name.endsWith('-skill.md')) {
        files.push(fullPath);
      }
    }

    return files;
  }

  /**
   * Load a skill from a markdown file
   * @param {string} filePath - Path to the skill markdown file
   * @returns {MarkdownSkill} Loaded skill instance
   */
  static loadSkill(filePath) {
    const content = readFileSync(filePath, 'utf8');
    return new MarkdownSkill(filePath, content);
  }

  /**
   * Load all skills from a directory
   * @param {string} baseDir - Base directory containing specs/
   * @returns {Array<MarkdownSkill>} Array of loaded skills
   */
  static loadAllSkills(baseDir = process.cwd()) {
    const skillFiles = this.scanForSkills(baseDir);
    return skillFiles.map(file => this.loadSkill(file));
  }
}

/**
 * Represents a skill loaded from markdown
 */
export class MarkdownSkill extends Skill {
  constructor(filePath, content) {
    super();
    this.filePath = filePath;
    this.content = content;
    this.parsed = this.parseMarkdown(content);
  }

  /**
   * Parse markdown skill file
   * @param {string} content - Markdown content
   * @returns {Object} Parsed skill data
   */
  parseMarkdown(content) {
    const lines = content.split('\n');
    const sections = {};
    let currentSection = null;
    let currentContent = [];

    // Parse frontmatter
    let frontmatter = {};
    let inFrontmatter = false;
    let frontmatterLines = [];

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];

      // Detect frontmatter
      if (line.trim() === '---') {
        if (!inFrontmatter && i === 0) {
          inFrontmatter = true;
          continue;
        } else if (inFrontmatter) {
          inFrontmatter = false;
          // Parse frontmatter
          frontmatterLines.forEach(fmLine => {
            const match = fmLine.match(/^(\w+):\s*(.+)$/);
            if (match) {
              frontmatter[match[1]] = match[2];
            }
          });
          continue;
        }
      }

      if (inFrontmatter) {
        frontmatterLines.push(line);
        continue;
      }

      // Parse sections
      if (line.startsWith('# ')) {
        // Save previous section
        if (currentSection) {
          sections[currentSection] = currentContent.join('\n').trim();
        }
        // Start new section - the title
        sections.title = line.substring(2).trim();
        currentSection = null;
        currentContent = [];
      } else if (line.startsWith('## ')) {
        // Save previous section
        if (currentSection) {
          sections[currentSection] = currentContent.join('\n').trim();
        }
        // Start new section
        currentSection = line.substring(3).trim().toLowerCase().replace(/\s+/g, '-');
        currentContent = [];
      } else {
        currentContent.push(line);
      }
    }

    // Save last section
    if (currentSection) {
      sections[currentSection] = currentContent.join('\n').trim();
    }

    return { frontmatter, sections };
  }

  /**
   * Extract triggers from markdown section
   * @returns {Array<string>} Array of trigger keywords
   */
  getTriggers() {
    const triggersSection = this.parsed.sections.triggers || '';
    const triggers = [];
    
    const lines = triggersSection.split('\n');
    for (const line of lines) {
      const trimmed = line.trim();
      // Match bullet points or quoted strings
      if (trimmed.startsWith('- ')) {
        const trigger = trimmed.substring(2).replace(/^["']|["']$/g, '').trim();
        if (trigger) {
          triggers.push(trigger);
        }
      }
    }
    
    return triggers;
  }

  /**
   * Extract workflow steps
   * @returns {Array<string>} Array of workflow steps
   */
  getWorkflow() {
    const workflowSection = this.parsed.sections.workflow || '';
    const steps = [];
    
    const lines = workflowSection.split('\n');
    for (const line of lines) {
      const trimmed = line.trim();
      // Match numbered steps
      const match = trimmed.match(/^\d+\.\s+(.+)$/);
      if (match) {
        steps.push(match[1]);
      }
    }
    
    return steps;
  }

  /**
   * Extract validation criteria
   * @returns {Array<string>} Array of validation criteria
   */
  getValidationCriteria() {
    const validationSection = this.parsed.sections.validation || '';
    const criteria = [];
    
    const lines = validationSection.split('\n');
    for (const line of lines) {
      const trimmed = line.trim();
      if (trimmed.startsWith('- ')) {
        criteria.push(trimmed.substring(2).trim());
      }
    }
    
    return criteria;
  }

  // Skill interface implementation

  getId() {
    // Use frontmatter id if available, otherwise derive from filename
    if (this.parsed.frontmatter.id) {
      return this.parsed.frontmatter.id;
    }
    
    const filename = this.filePath.split('/').pop();
    return filename.replace('-skill.md', '');
  }

  getName() {
    return this.parsed.sections.title || this.getId();
  }

  getDescription() {
    // Use the content before the first ## section as description
    const description = this.parsed.sections.description || '';
    if (description) {
      return description;
    }

    // Fallback: use first paragraph after title
    const content = this.content;
    const lines = content.split('\n');
    let afterTitle = false;
    let desc = [];

    for (const line of lines) {
      if (line.startsWith('# ')) {
        afterTitle = true;
        continue;
      }
      if (afterTitle && line.startsWith('## ')) {
        break;
      }
      if (afterTitle && line.trim()) {
        desc.push(line.trim());
      }
    }

    return desc.join(' ').substring(0, 200);
  }

  shouldTrigger(userInput, context = {}) {
    const triggers = this.getTriggers();
    const lowerInput = userInput.toLowerCase();
    
    return triggers.some(trigger => 
      lowerInput.includes(trigger.toLowerCase())
    );
  }

  getQuestions() {
    // Markdown skills don't use the question-based interface
    // They are executed directly by the coach's intelligence
    return [];
  }

  processResponses(responses) {
    // Markdown skills are executed by the coach, not through structured responses
    // Return workflow steps for the coach to execute
    return {
      summary: `Execute workflow for ${this.getName()}`,
      workflow: this.getWorkflow(),
      validation: this.getValidationCriteria(),
      data: {
        skillId: this.getId(),
        skillName: this.getName()
      }
    };
  }

  /**
   * Get workflow steps for coach execution
   * @returns {Object} Workflow execution data
   */
  getExecutionData() {
    return {
      skillId: this.getId(),
      skillName: this.getName(),
      description: this.getDescription(),
      workflow: this.getWorkflow(),
      validation: this.getValidationCriteria(),
      triggers: this.getTriggers()
    };
  }
}
