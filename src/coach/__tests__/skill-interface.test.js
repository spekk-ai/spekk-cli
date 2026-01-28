import { test, describe } from 'node:test';
import assert from 'node:assert';
import { Skill } from '../skill-interface.js';

describe('Skill Interface', () => {
  class TestSkill extends Skill {
    getId() { return 'test-skill'; }
    getName() { return 'Test Skill'; }
    getDescription() { return 'A test skill'; }
    shouldTrigger(userInput) { return userInput.includes('test'); }
    getQuestions() { 
      return [
        { id: 'q1', text: 'Question 1?', type: 'text' },
        { id: 'q2', text: 'Question 2?', type: 'choice', options: ['A', 'B'] }
      ]; 
    }
    processResponses(responses) {
      return { 
        summary: 'Test summary',
        recommendations: ['Test recommendation'],
        data: responses 
      };
    }
  }

  describe('Base Skill class', () => {
    test('should throw errors for unimplemented methods', () => {
      const skill = new Skill();
      
      assert.throws(() => skill.getId(), /Skill must implement getId/);
      assert.throws(() => skill.getName(), /Skill must implement getName/);
      assert.throws(() => skill.getDescription(), /Skill must implement getDescription/);
      assert.throws(() => skill.shouldTrigger(), /Skill must implement shouldTrigger/);
      assert.throws(() => skill.getQuestions(), /Skill must implement getQuestions/);
      assert.throws(() => skill.processResponses(), /Skill must implement processResponses/);
    });

    test('should provide default output format', () => {
      const skill = new TestSkill();
      const format = skill.getOutputFormat();
      
      assert.deepStrictEqual(format, {
        filePattern: '{projectName}/test-skill-output.md',
        format: 'markdown',
        sections: ['summary', 'recommendations', 'data']
      });
    });

    test('should validate responses', () => {
      const skill = new TestSkill();
      
      // Test with missing responses
      const validation1 = skill.validateResponses({ q1: 'Answer 1' });
      assert.strictEqual(validation1.valid, false);
      assert(validation1.errors.includes('Missing response for: Question 2?'));
      
      // Test with complete responses
      const validation2 = skill.validateResponses({ q1: 'Answer 1', q2: 'A' });
      assert.strictEqual(validation2.valid, true);
      assert.strictEqual(validation2.errors.length, 0);
    });
  });

  describe('Skill implementation', () => {
    test('should implement all required methods', () => {
      const skill = new TestSkill();
      
      assert.strictEqual(skill.getId(), 'test-skill');
      assert.strictEqual(skill.getName(), 'Test Skill');
      assert.strictEqual(skill.getDescription(), 'A test skill');
      assert.strictEqual(skill.shouldTrigger('test input'), true);
      assert.strictEqual(skill.shouldTrigger('other input'), false);
      
      const questions = skill.getQuestions();
      assert.strictEqual(questions.length, 2);
      assert.strictEqual(questions[0].id, 'q1');
      assert.strictEqual(questions[1].type, 'choice');
      
      const result = skill.processResponses({ q1: 'A1', q2: 'B' });
      assert.strictEqual(result.summary, 'Test summary');
      assert.deepStrictEqual(result.recommendations, ['Test recommendation']);
      assert.deepStrictEqual(result.data, { q1: 'A1', q2: 'B' });
    });
  });
});