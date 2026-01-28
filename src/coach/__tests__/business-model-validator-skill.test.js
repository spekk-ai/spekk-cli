import { test, describe, beforeEach } from 'node:test';
import assert from 'node:assert';
import { BusinessModelValidator } from '../business-model-validator.js';
import { Skill } from '../skill-interface.js';

describe('BusinessModelValidator as Skill', () => {
  let validator;
  
  beforeEach(() => {
    validator = new BusinessModelValidator();
  });

  test('should extend Skill class', () => {
    assert(validator instanceof Skill);
  });

  describe('Skill interface implementation', () => {
    test('should implement getId', () => {
      assert.strictEqual(validator.getId(), 'business-model-validator');
    });

    test('should implement getName', () => {
      assert.strictEqual(validator.getName(), 'Business Model Validator');
    });

    test('should implement getDescription', () => {
      const description = validator.getDescription();
      assert(description.includes('Validates startup/business models'));
      assert(description.includes('structured questioning'));
    });

    test('should implement shouldTrigger', () => {
      // Should trigger for business-related keywords
      assert.strictEqual(validator.shouldTrigger('I have a startup idea'), true);
      assert.strictEqual(validator.shouldTrigger('validate my business model'), true);
      assert.strictEqual(validator.shouldTrigger('what is my revenue model'), true);
      assert.strictEqual(validator.shouldTrigger('market validation for my product'), true);
      
      // Should not trigger for unrelated input
      assert.strictEqual(validator.shouldTrigger('what is the weather today'), false);
      assert.strictEqual(validator.shouldTrigger('write me a poem'), false);
    });

    test('should implement getQuestions', () => {
      const questions = validator.getQuestions();
      
      // Should have questions for all 6 areas with 4 questions each
      assert.strictEqual(questions.length, 24); // 6 areas * 4 questions
      
      // Check question structure
      const firstQuestion = questions[0];
      assert('id' in firstQuestion);
      assert('text' in firstQuestion);
      assert('type' in firstQuestion);
      assert('area' in firstQuestion);
      
      // Check that all areas are covered
      const areas = new Set(questions.map(q => q.area));
      assert.strictEqual(areas.size, 6);
      assert.strictEqual(areas.has('industry-background'), true);
      assert.strictEqual(areas.has('product-validation'), true);
    });

    test('should implement processResponses', () => {
      // Create mock responses for all questions
      const mockResponses = {};
      const questions = validator.getQuestions();
      
      questions.forEach(q => {
        mockResponses[q.id] = 'This is a detailed response that should score well.';
      });
      
      const result = validator.processResponses(mockResponses);
      
      // Check result structure
      assert('summary' in result);
      assert('recommendations' in result);
      assert('data' in result);
      assert('markdownReport' in result);
      
      // Check data structure
      assert('totalScore' in result.data);
      assert('percentageScore' in result.data);
      assert('areaScores' in result.data);
      assert('responses' in result.data);
      
      // Should have recommendations array
      assert(Array.isArray(result.recommendations));
      
      // Markdown report should contain expected sections
      assert(result.markdownReport.includes('BUSINESS MODEL VALIDATION REPORT'));
      assert(result.markdownReport.includes('Overall Health Score'));
      assert(result.markdownReport.includes('Strengths:'));
      assert(result.markdownReport.includes('Concerns:'));
    });

    test('should validate responses using Skill base class method', () => {
      const questions = validator.getQuestions();
      
      // Test with incomplete responses
      const partialResponses = {};
      questions.slice(0, 5).forEach(q => {
        partialResponses[q.id] = 'Response';
      });
      
      const validation1 = validator.validateResponses(partialResponses);
      assert.strictEqual(validation1.valid, false);
      assert(validation1.errors.length > 0);
      
      // Test with complete responses
      const completeResponses = {};
      questions.forEach(q => {
        completeResponses[q.id] = 'Response';
      });
      
      const validation2 = validator.validateResponses(completeResponses);
      assert.strictEqual(validation2.valid, true);
      assert.strictEqual(validation2.errors.length, 0);
    });
  });

  describe('Auto-scoring logic', () => {
    test('should score based on response length', () => {
      const shortResponses = ['Yes', 'No', 'Maybe'];
      assert.strictEqual(validator.autoScore(shortResponses), 0);
      
      const mediumResponses = ['This is a medium length response that provides some detail.'];
      assert.strictEqual(validator.autoScore(mediumResponses), 1);
      
      const longResponses = ['This is a very detailed response that provides comprehensive information about the topic at hand, including multiple perspectives and thorough analysis of the situation.'];
      assert.strictEqual(validator.autoScore(longResponses), 2);
    });
  });
});