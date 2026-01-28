import { Skill } from './skill-interface.js';

/**
 * Central registry for all coach skills
 * Manages skill registration, detection, and execution
 */
export class SkillRegistry {
  constructor() {
    this.skills = new Map();
  }

  /**
   * Register a new skill
   * @param {Skill} skill - The skill instance to register
   */
  register(skill) {
    if (!(skill instanceof Skill)) {
      throw new Error('Skill must extend the Skill base class');
    }

    const id = skill.getId();
    if (this.skills.has(id)) {
      throw new Error(`Skill with ID '${id}' is already registered`);
    }

    this.skills.set(id, skill);
  }

  /**
   * Unregister a skill
   * @param {string} skillId - The ID of the skill to unregister
   */
  unregister(skillId) {
    this.skills.delete(skillId);
  }

  /**
   * Get a skill by ID
   * @param {string} skillId - The skill ID
   * @returns {Skill|undefined} The skill instance or undefined
   */
  getSkill(skillId) {
    return this.skills.get(skillId);
  }

  /**
   * Get all registered skills
   * @returns {Array<Skill>} Array of all registered skills
   */
  getAllSkills() {
    return Array.from(this.skills.values());
  }

  /**
   * Detect which skills should be triggered based on user input
   * @param {string} userInput - The user's input text
   * @param {Object} context - Additional context
   * @returns {Array<Skill>} Array of skills that should be triggered
   */
  detectSkills(userInput, context = {}) {
    const triggeredSkills = [];

    for (const skill of this.skills.values()) {
      if (skill.shouldTrigger(userInput, context)) {
        triggeredSkills.push(skill);
      }
    }

    return triggeredSkills;
  }

  /**
   * Get skill suggestions as formatted strings
   * @param {string} userInput - The user's input text
   * @param {Object} context - Additional context
   * @returns {Array<Object>} Array of skill suggestions with { id, name, description }
   */
  getSuggestions(userInput, context = {}) {
    const triggeredSkills = this.detectSkills(userInput, context);

    return triggeredSkills.map(skill => ({
      id: skill.getId(),
      name: skill.getName(),
      description: skill.getDescription()
    }));
  }

  /**
   * Create a skill session for interactive execution
   * @param {string} skillId - The skill to execute
   * @returns {SkillSession} A session object for managing skill execution
   */
  createSession(skillId) {
    const skill = this.getSkill(skillId);
    if (!skill) {
      throw new Error(`Skill '${skillId}' not found`);
    }

    return new SkillSession(skill);
  }
}

/**
 * Manages the execution of a single skill session
 */
export class SkillSession {
  constructor(skill) {
    this.skill = skill;
    this.responses = {};
    this.currentQuestionIndex = 0;
    this.startedAt = new Date().toISOString();
    this.completedAt = null;
  }

  /**
   * Get the current question
   * @returns {Object|null} Current question or null if complete
   */
  getCurrentQuestion() {
    const questions = this.skill.getQuestions();
    if (this.currentQuestionIndex >= questions.length) {
      return null;
    }
    return questions[this.currentQuestionIndex];
  }

  /**
   * Record a response and move to next question
   * @param {string} questionId - The question ID
   * @param {any} response - The user's response
   */
  recordResponse(questionId, response) {
    this.responses[questionId] = response;
    this.currentQuestionIndex++;
  }

  /**
   * Check if all questions have been answered
   * @returns {boolean} Whether the session is complete
   */
  isComplete() {
    const questions = this.skill.getQuestions();
    return this.currentQuestionIndex >= questions.length;
  }

  /**
   * Get progress as a percentage
   * @returns {number} Progress percentage (0-100)
   */
  getProgress() {
    const questions = this.skill.getQuestions();
    if (questions.length === 0) return 100;
    return Math.round((this.currentQuestionIndex / questions.length) * 100);
  }

  /**
   * Validate all responses
   * @returns {Object} Validation result
   */
  validate() {
    return this.skill.validateResponses(this.responses);
  }

  /**
   * Process responses and generate output
   * @returns {Object} Structured output from the skill
   */
  process() {
    if (!this.isComplete()) {
      throw new Error('Session is not complete');
    }

    const validation = this.validate();
    if (!validation.valid) {
      throw new Error(`Invalid responses: ${validation.errors.join(', ')}`);
    }

    this.completedAt = new Date().toISOString();
    return this.skill.processResponses(this.responses);
  }

  /**
   * Get session metadata
   * @returns {Object} Session metadata
   */
  getMetadata() {
    return {
      skillId: this.skill.getId(),
      skillName: this.skill.getName(),
      startedAt: this.startedAt,
      completedAt: this.completedAt,
      progress: this.getProgress(),
      responseCount: Object.keys(this.responses).length
    };
  }
}

// Create and export a singleton instance
export const skillRegistry = new SkillRegistry();