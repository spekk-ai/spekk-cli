import { test, describe, beforeEach } from 'node:test';
import assert from 'node:assert';
import { SkillRegistry, SkillSession } from '../skill-registry.js';
import { Skill } from '../skill-interface.js';

// Mock skill implementations
class MockSkill1 extends Skill {
  getId() { return 'mock-skill-1'; }
  getName() { return 'Mock Skill 1'; }
  getDescription() { return 'First mock skill'; }
  shouldTrigger(userInput) { return userInput.includes('mock1'); }
  getQuestions() { 
    return [
      { id: 'q1', text: 'Question 1?', type: 'text' },
      { id: 'q2', text: 'Question 2?', type: 'text' }
    ]; 
  }
  processResponses(responses) {
    return { summary: 'Mock1 processed', recommendations: [], data: responses };
  }
}

class MockSkill2 extends Skill {
  getId() { return 'mock-skill-2'; }
  getName() { return 'Mock Skill 2'; }
  getDescription() { return 'Second mock skill'; }
  shouldTrigger(userInput) { return userInput.includes('mock2'); }
  getQuestions() { 
    return [{ id: 'q1', text: 'Single question?', type: 'text' }]; 
  }
  processResponses(responses) {
    return { summary: 'Mock2 processed', recommendations: [], data: responses };
  }
}

describe('SkillRegistry', () => {
  let registry;
  
  beforeEach(() => {
    registry = new SkillRegistry();
  });

  describe('register', () => {
    test('should register a skill', () => {
      const skill = new MockSkill1();
      registry.register(skill);
      
      assert.strictEqual(registry.getSkill('mock-skill-1'), skill);
    });

    test('should throw error for invalid skill', () => {
      assert.throws(() => registry.register({}), /Skill must extend the Skill base class/);
    });

    test('should throw error for duplicate skill ID', () => {
      const skill = new MockSkill1();
      registry.register(skill);
      
      assert.throws(() => registry.register(skill), /Skill with ID 'mock-skill-1' is already registered/);
    });
  });

  describe('unregister', () => {
    test('should unregister a skill', () => {
      const skill = new MockSkill1();
      registry.register(skill);
      registry.unregister('mock-skill-1');
      
      assert.strictEqual(registry.getSkill('mock-skill-1'), undefined);
    });
  });

  describe('getAllSkills', () => {
    test('should return all registered skills', () => {
      const skill1 = new MockSkill1();
      const skill2 = new MockSkill2();
      
      registry.register(skill1);
      registry.register(skill2);
      
      const skills = registry.getAllSkills();
      assert.strictEqual(skills.length, 2);
      assert(skills.includes(skill1));
      assert(skills.includes(skill2));
    });
  });

  describe('detectSkills', () => {
    beforeEach(() => {
      registry.register(new MockSkill1());
      registry.register(new MockSkill2());
    });

    test('should detect skills based on trigger conditions', () => {
      const skills1 = registry.detectSkills('I need help with mock1');
      assert.strictEqual(skills1.length, 1);
      assert.strictEqual(skills1[0].getId(), 'mock-skill-1');
      
      const skills2 = registry.detectSkills('Tell me about mock2');
      assert(skills2).toHaveLength(1);
      assert(skills2[0].getId()).toBe('mock-skill-2');
      
      const skills3 = registry.detectSkills('mock1 and mock2 together');
      assert(skills3).toHaveLength(2);
    });

    test('should return empty array when no skills trigger', () => {
      const skills = registry.detectSkills('unrelated input');
      assert(skills).toHaveLength(0);
    });
  });

  describe('getSuggestions', () => {
    beforeEach(() => {
      registry.register(new MockSkill1());
      registry.register(new MockSkill2());
    });

    test('should return formatted skill suggestions', () => {
      const suggestions = registry.getSuggestions('I need mock1 help');
      
      assert(suggestions).toHaveLength(1);
      assert(suggestions[0]).toEqual({
        id: 'mock-skill-1',
        name: 'Mock Skill 1',
        description: 'First mock skill'
      });
    });
  });

  describe('createSession', () => {
    test('should create a session for existing skill', () => {
      const skill = new MockSkill1();
      registry.register(skill);
      
      const session = registry.createSession('mock-skill-1');
      assert(session).toBeInstanceOf(SkillSession);
      assert(session.skill).toBe(skill);
    });

    test('should throw error for non-existent skill', () => {
      assert(() => registry.createSession('unknown-skill')).toThrow("Skill 'unknown-skill' not found");
    });
  });
});

describe('SkillSession', () => {
  let session;
  let skill;
  
  beforeEach(() => {
    skill = new MockSkill1();
    session = new SkillSession(skill);
  });

  describe('getCurrentQuestion', () => {
    test('should return current question', () => {
      const question = session.getCurrentQuestion();
      assert(question.id).toBe('q1');
      assert(question.text).toBe('Question 1?');
    });

    test('should return null when complete', () => {
      session.recordResponse('q1', 'Answer 1');
      session.recordResponse('q2', 'Answer 2');
      
      assert(session.getCurrentQuestion()).toBeNull();
    });
  });

  describe('recordResponse', () => {
    test('should record response and advance to next question', () => {
      session.recordResponse('q1', 'Answer 1');
      
      assert(session.responses.q1).toBe('Answer 1');
      assert(session.currentQuestionIndex).toBe(1);
      assert(session.getCurrentQuestion().id).toBe('q2');
    });
  });

  describe('isComplete', () => {
    test('should return false when questions remain', () => {
      assert.strictEqual(session.isComplete(), false);
      
      session.recordResponse('q1', 'Answer 1');
      assert.strictEqual(session.isComplete(), false);
    });

    test('should return true when all questions answered', () => {
      session.recordResponse('q1', 'Answer 1');
      session.recordResponse('q2', 'Answer 2');
      
      assert(session.isComplete()).toBe(true);
    });
  });

  describe('getProgress', () => {
    test('should calculate progress percentage', () => {
      assert(session.getProgress()).toBe(0);
      
      session.recordResponse('q1', 'Answer 1');
      assert(session.getProgress()).toBe(50);
      
      session.recordResponse('q2', 'Answer 2');
      assert(session.getProgress()).toBe(100);
    });
  });

  describe('validate', () => {
    test('should validate responses', () => {
      session.recordResponse('q1', 'Answer 1');
      
      const validation1 = session.validate();
      assert(validation1.valid).toBe(false);
      
      session.recordResponse('q2', 'Answer 2');
      const validation2 = session.validate();
      assert(validation2.valid).toBe(true);
    });
  });

  describe('process', () => {
    test('should process complete session', () => {
      session.recordResponse('q1', 'Answer 1');
      session.recordResponse('q2', 'Answer 2');
      
      const result = session.process();
      assert(result.summary).toBe('Mock1 processed');
      assert(result.data).toEqual({ q1: 'Answer 1', q2: 'Answer 2' });
      assert(session.completedAt).toBeTruthy();
    });

    test('should throw error for incomplete session', () => {
      session.recordResponse('q1', 'Answer 1');
      
      assert(() => session.process()).toThrow('Session is not complete');
    });
  });

  describe('getMetadata', () => {
    test('should return session metadata', () => {
      session.recordResponse('q1', 'Answer 1');
      
      const metadata = session.getMetadata();
      assert(metadata.skillId).toBe('mock-skill-1');
      assert(metadata.skillName).toBe('Mock Skill 1');
      assert(metadata.progress).toBe(50);
      assert(metadata.responseCount).toBe(1);
      assert(metadata.startedAt).toBeTruthy();
      assert(metadata.completedAt).toBeNull();
    });
  });
});