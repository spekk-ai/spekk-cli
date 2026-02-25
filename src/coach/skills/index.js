/**
 * Coach Skills Framework
 * Central module for managing all coach skills
 * Now uses markdown-based skills instead of JavaScript classes
 */

import { skillRegistry } from '../skill-registry.js';
import { MarkdownSkillLoader } from '../markdown-skill-loader.js';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

// Get project root directory
const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const projectRoot = resolve(__dirname, '../../..');

// Register all available skills from markdown files
export function registerAllSkills() {
  const skills = MarkdownSkillLoader.loadAllSkills(projectRoot);
  
  skills.forEach(skill => {
    skillRegistry.register(skill);
  });
  
  console.log(`Loaded ${skills.length} markdown skill(s) from specs/coach-skills-system/`);
}

// Initialize skills on module load
registerAllSkills();

// Export the registry and useful utilities
export { skillRegistry };
export { Skill } from '../skill-interface.js';
export { SkillSession } from '../skill-registry.js';
export { MarkdownSkillLoader, MarkdownSkill } from '../markdown-skill-loader.js';

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

/**
 * Helper function to get workflow execution data for a skill
 * Useful for coach agents to understand how to execute the skill
 * @param {string} skillId - The skill ID
 * @returns {Object|null} Execution data or null if skill not found
 */
export function getSkillWorkflow(skillId) {
  const skill = skillRegistry.getSkill(skillId);
  if (!skill) return null;
  
  // If it's a markdown skill, get execution data
  if (skill.getExecutionData) {
    return skill.getExecutionData();
  }
  
  // Fallback for non-markdown skills
  return {
    skillId: skill.getId(),
    skillName: skill.getName(),
    description: skill.getDescription()
  };
}