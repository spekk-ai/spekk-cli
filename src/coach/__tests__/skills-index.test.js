import { test, describe, beforeEach } from 'node:test';
import assert from 'node:assert';
import { 
  skillRegistry, 
  detectSkills, 
  createSkillSession, 
  listAvailableSkills 
} from '../skills/index.js';
import { BusinessModelValidator } from '../business-model-validator.js';

describe('Skills Framework Index', () => {
  // Clear registry before each test to ensure clean state
  beforeEach(() => {
    // Unregister all skills
    const skills = skillRegistry.getAllSkills();
    skills.forEach(skill => {
      skillRegistry.unregister(skill.getId());
    });
    
    // Re-register default skills
    skillRegistry.register(new BusinessModelValidator());
  });

  describe('Default skill registration', () => {
    test('should automatically register BusinessModelValidator', () => {
      const skill = skillRegistry.getSkill('business-model-validator');
      assert(skill);
      assert(skill instanceof BusinessModelValidator);
    });
  });

  describe('detectSkills helper', () => {
    test('should detect skills based on user input', () => {
      const suggestions = detectSkills('I need help validating my startup idea');
      
      assert.strictEqual(suggestions.length, 1);
      assert.strictEqual(suggestions[0].id, 'business-model-validator');
      assert.strictEqual(suggestions[0].name, 'Business Model Validator');
    });

    test('should return empty array for unrelated input', () => {
      const suggestions = detectSkills('What is the capital of France?');
      assert.strictEqual(suggestions.length, 0);
    });
  });

  describe('createSkillSession helper', () => {
    test('should create a session for existing skill', () => {
      const session = createSkillSession('business-model-validator');
      
      assert(session);
      assert.strictEqual(session.skill.getId(), 'business-model-validator');
      assert.strictEqual(session.isComplete(), false);
    });

    test('should throw error for non-existent skill', () => {
      assert.throws(() => createSkillSession('non-existent-skill'),
        /Skill 'non-existent-skill' not found/);
    });
  });

  describe('listAvailableSkills helper', () => {
    test('should list all registered skills', () => {
      const skills = listAvailableSkills();
      
      assert.strictEqual(skills.length, 1);
      assert.strictEqual(skills[0].id, 'business-model-validator');
      assert.strictEqual(skills[0].name, 'Business Model Validator');
      assert(skills[0].description.includes('Validates startup/business models'));
    });
  });

  describe('Extensibility', () => {
    test('should support registering new skills', () => {
      // Mock a new skill
      class TestSkill {
        constructor() {}
        getId() { return 'test-skill'; }
        getName() { return 'Test Skill'; }
        getDescription() { return 'A test skill'; }
        shouldTrigger(input) { return input.includes('test'); }
        getQuestions() { return []; }
        processResponses() { return { summary: 'Test', recommendations: [], data: {} }; }
        getOutputFormat() { return { filePattern: 'test.md', format: 'markdown', sections: [] }; }
        validateResponses() { return { valid: true, errors: [] }; }
      }
      
      // Extend from Skill base class
      const { Skill } = await import('../skill-interface.js');
      Object.setPrototypeOf(TestSkill.prototype, Skill.prototype);
      
      // Register the new skill
      const testSkill = new TestSkill();
      skillRegistry.register(testSkill);
      
      // Verify it's registered
      const skills = listAvailableSkills();
      assert.strictEqual(skills.length, 2);
      assert(skills.find(s => s.id === 'test-skill'));
      
      // Verify detection works
      const suggestions = detectSkills('I need a test');
      assert(suggestions.find(s => s.id === 'test-skill'));
    });
  });
});