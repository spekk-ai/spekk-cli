/**
 * Base interface for all coach skills
 * Every skill must implement this interface to work with the skill framework
 */
export class Skill {
  /**
   * Unique identifier for the skill
   * @returns {string} The skill ID
   */
  getId() {
    throw new Error('Skill must implement getId()');
  }

  /**
   * Human-readable name for the skill
   * @returns {string} The skill name
   */
  getName() {
    throw new Error('Skill must implement getName()');
  }

  /**
   * Description of what this skill does
   * @returns {string} The skill description
   */
  getDescription() {
    throw new Error('Skill must implement getDescription()');
  }

  /**
   * Check if this skill should be triggered based on user input or context
   * @param {string} userInput - The user's input text
   * @param {Object} context - Additional context (e.g., conversation history)
   * @returns {boolean} Whether this skill should be suggested
   */
  shouldTrigger(userInput, context = {}) {
    throw new Error('Skill must implement shouldTrigger()');
  }

  /**
   * Get structured questions for this skill
   * @returns {Array<Object>} Array of question objects with structure:
   *   { id: string, text: string, type: 'text'|'choice'|'scale', options?: Array }
   */
  getQuestions() {
    throw new Error('Skill must implement getQuestions()');
  }

  /**
   * Process responses and generate structured output
   * @param {Object} responses - User responses to questions
   * @returns {Object} Structured output with at minimum: { summary, recommendations, data }
   */
  processResponses(responses) {
    throw new Error('Skill must implement processResponses()');
  }

  /**
   * Get the output format specification for this skill
   * @returns {Object} Format specification including file conventions
   */
  getOutputFormat() {
    return {
      filePattern: `{projectName}/${this.getId()}-output.md`,
      format: 'markdown',
      sections: ['summary', 'recommendations', 'data']
    };
  }

  /**
   * Validate that responses are complete and valid
   * @param {Object} responses - User responses to validate
   * @returns {Object} Validation result: { valid: boolean, errors?: Array<string> }
   */
  validateResponses(responses) {
    const questions = this.getQuestions();
    const errors = [];

    questions.forEach(question => {
      if (!responses[question.id]) {
        errors.push(`Missing response for: ${question.text}`);
      }
    });

    return {
      valid: errors.length === 0,
      errors
    };
  }
}