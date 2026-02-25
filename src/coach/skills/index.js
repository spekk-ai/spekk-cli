/**
 * Coach Skills Framework
 * Central module for managing all coach skills
 */

import { skillRegistry } from '../skill-registry.js';
import { BusinessModelValidator } from '../business-model-validator.js';
import { MeetingNotesToSpecs } from '../meeting-notes-to-specs.js';
import { Coordinator } from '../coordinator.js';

// Register all available skills
export function registerAllSkills() {
  // Business Model Validator
  skillRegistry.register(new BusinessModelValidator());

  // Meeting Notes to Specs
  skillRegistry.register(new MeetingNotesToSpecs());

  // Coordinator
  skillRegistry.register(new Coordinator());
}

// Initialize skills on module load
registerAllSkills();

// Export the registry and useful utilities
export { skillRegistry };
export { Skill } from '../skill-interface.js';
export { SkillSession } from '../skill-registry.js';

/**
 * Helper function to detect and suggest skills based on user input
 * @param {string} userInput - The user's input
 * @param {Object} context - Additional context
 * @returns {Array} Array of skill suggestions
 */
export function detectSkills(userInput, context = {}) {
  return skillRegistry.getSuggestions(userInput, context);
}

/**
 * Helper function to create a skill session
 * @param {string} skillId - The skill ID
 * @returns {SkillSession} The skill session
 */
export function createSkillSession(skillId) {
  return skillRegistry.createSession(skillId);
}

/**
 * Helper function to list all available skills
 * @returns {Array} Array of skill info objects
 */
export function listAvailableSkills() {
  return skillRegistry.getAllSkills().map(skill => ({
    id: skill.getId(),
    name: skill.getName(),
    description: skill.getDescription()
  }));
}