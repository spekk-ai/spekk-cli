import { test, describe, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert';
import fs from 'node:fs';
import path from 'node:path';
import { BusinessModelValidator } from '../business-model-validator.js';

describe('BusinessModelValidator', () => {
  let validator;

  beforeEach(() => {
    validator = new BusinessModelValidator();
  });

  describe('assessment areas', () => {
    test('has 6 assessment areas', () => {
      const areas = validator.getAssessmentAreas();
      assert.strictEqual(areas.length, 6);
      assert.deepStrictEqual(areas, [
        'industry-background',
        'product-validation', 
        'market-demand',
        'business-goals',
        'traction-momentum',
        'hypothesis-prioritization'
      ]);
    });

    test('each area has sample questions', () => {
      const areas = validator.getAssessmentAreas();
      areas.forEach(area => {
        const questions = validator.getQuestionsForArea(area);
        assert.ok(questions !== undefined);
        assert.ok(questions.length > 0);
        questions.forEach(question => {
          assert.strictEqual(typeof question, 'string');
          assert.ok(question.length > 0);
        });
      });
    });
  });

  describe('scoring methodology', () => {
    test('validates score range', () => {
      assert.strictEqual(validator.isValidScore(0), true);
      assert.strictEqual(validator.isValidScore(1), true);
      assert.strictEqual(validator.isValidScore(2), true);
      assert.strictEqual(validator.isValidScore(-1), false);
      assert.strictEqual(validator.isValidScore(3), false);
      assert.strictEqual(validator.isValidScore('invalid'), false);
    });

    test('calculates total score correctly', () => {
      const scores = {
        'industry-background': 2,
        'product-validation': 1,
        'market-demand': 0,
        'business-goals': 2,
        'traction-momentum': 1,
        'hypothesis-prioritization': 2
      };
      
      assert.strictEqual(validator.calculateTotalScore(scores), 8);
    });

    test('calculates percentage score', () => {
      const scores = {
        'industry-background': 2,
        'product-validation': 1,
        'market-demand': 0,
        'business-goals': 2,
        'traction-momentum': 1,
        'hypothesis-prioritization': 2
      };
      
      const percentage = validator.calculatePercentageScore(scores);
      assert.strictEqual(percentage, 67); // 8/12 * 100 = 66.67, rounded to 67
    });
  });

  describe('assessment session', () => {
    test('initializes with empty responses', () => {
      const session = validator.createSession();
      assert.deepStrictEqual(session.responses, {});
      assert.deepStrictEqual(session.scores, {});
      assert.strictEqual(session.currentArea, null);
      assert.strictEqual(session.isComplete, false);
    });

    test('records response for area', () => {
      const session = validator.createSession();
      validator.recordResponse(session, 'industry-background', 'Founder has 10 years experience in fintech');
      
      assert.strictEqual(session.responses['industry-background'], 'Founder has 10 years experience in fintech');
    });

    test('records score for area', () => {
      const session = validator.createSession();
      validator.recordScore(session, 'industry-background', 2);
      
      assert.strictEqual(session.scores['industry-background'], 2);
    });

    test('identifies complete session', () => {
      const session = validator.createSession();
      const areas = validator.getAssessmentAreas();
      
      // Score all areas
      areas.forEach(area => {
        validator.recordScore(session, area, 2);
      });
      
      assert.strictEqual(validator.isSessionComplete(session), true);
    });

    test('identifies incomplete session', () => {
      const session = validator.createSession();
      validator.recordScore(session, 'industry-background', 2);
      
      assert.strictEqual(validator.isSessionComplete(session), false);
    });
  });

  describe('recommendations', () => {
    test('generates recommendations based on scores', () => {
      const scores = {
        'industry-background': 0,
        'product-validation': 1,
        'market-demand': 2,
        'business-goals': 0,
        'traction-momentum': 1,
        'hypothesis-prioritization': 2
      };
      
      const recommendations = validator.generateRecommendations(scores);
      assert.ok(recommendations !== undefined);
      assert.strictEqual(Array.isArray(recommendations), true);
      assert.ok(recommendations.length > 0);
    });

    test('recommendations prioritize lowest scores', () => {
      const scores = {
        'industry-background': 0,
        'product-validation': 2,
        'market-demand': 2,
        'business-goals': 2,
        'traction-momentum': 2,
        'hypothesis-prioritization': 2
      };
      
      const recommendations = validator.generateRecommendations(scores);
      const firstRecommendation = recommendations[0];
      assert.strictEqual(firstRecommendation.area, 'industry-background');
      assert.strictEqual(firstRecommendation.priority, 'high');
    });
  });

  describe('report generation', () => {
    test('generates assessment report', () => {
      const session = validator.createSession();
      const areas = validator.getAssessmentAreas();
      
      // Complete the session
      areas.forEach((area, index) => {
        validator.recordResponse(session, area, `Response for ${area}`);
        validator.recordScore(session, area, index % 3); // Mix of scores 0,1,2
      });
      
      const report = validator.generateReport(session);
      
      assert.ok(report !== undefined);
      assert.ok(report.totalScore !== undefined);
      assert.ok(report.percentageScore !== undefined);
      assert.ok(report.areaScores !== undefined);
      assert.ok(report.recommendations !== undefined);
      assert.ok(report.summary !== undefined);
    });
  });

  describe('structured assessment report', () => {
    test('generates markdown report content', () => {
      const session = validator.createSession();
      const areas = validator.getAssessmentAreas();
      
      // Complete the session with mixed scores
      areas.forEach((area, index) => {
        validator.recordResponse(session, area, `Response for ${area}`);
        validator.recordScore(session, area, index % 3);
      });
      
      const markdownReport = validator.generateStructuredReport(session);
      
      assert.strictEqual(typeof markdownReport, 'string');
      assert.ok(markdownReport.includes('BUSINESS MODEL VALIDATION REPORT'));
      assert.ok(markdownReport.includes('Overall Health Score:'));
      assert.ok(markdownReport.includes('Strengths:'));
      assert.ok(markdownReport.includes('Concerns:'));
      assert.ok(markdownReport.includes('Unanswered/Unclear:'));
      assert.ok(markdownReport.includes('Priority Hypotheses to Test'));
      assert.ok(markdownReport.includes('Recommended Actions:'));
    });

    test('identifies strengths from high scoring areas', () => {
      const session = validator.createSession();
      const areas = validator.getAssessmentAreas();
      
      // Set specific scores - make some areas strong (score 2)
      validator.recordScore(session, 'industry-background', 2);
      validator.recordScore(session, 'product-validation', 2);
      validator.recordScore(session, 'market-demand', 0);
      validator.recordScore(session, 'business-goals', 1);
      validator.recordScore(session, 'traction-momentum', 0);
      validator.recordScore(session, 'hypothesis-prioritization', 1);
      
      const markdownReport = validator.generateStructuredReport(session);
      
      // Should identify strengths from score 2 areas
      assert.ok(markdownReport.includes('Industry Background & Expertise'));
      assert.ok(markdownReport.includes('Product Validation'));
    });

    test('identifies concerns from low scoring areas', () => {
      const session = validator.createSession();
      
      validator.recordScore(session, 'industry-background', 2);
      validator.recordScore(session, 'product-validation', 1);
      validator.recordScore(session, 'market-demand', 0);
      validator.recordScore(session, 'business-goals', 1);
      validator.recordScore(session, 'traction-momentum', 0);
      validator.recordScore(session, 'hypothesis-prioritization', 1);
      
      const markdownReport = validator.generateStructuredReport(session);
      
      // Should identify concerns from score 0 areas
      assert.ok(markdownReport.includes('Market Demand'));
      assert.ok(markdownReport.includes('Traction & Momentum'));
    });

    test('generates priority hypotheses from recommendations', () => {
      const session = validator.createSession();
      
      validator.recordScore(session, 'industry-background', 0);
      validator.recordScore(session, 'product-validation', 0);
      validator.recordScore(session, 'market-demand', 2);
      validator.recordScore(session, 'business-goals', 2);
      validator.recordScore(session, 'traction-momentum', 2);
      validator.recordScore(session, 'hypothesis-prioritization', 2);
      
      const markdownReport = validator.generateStructuredReport(session);
      
      // Should have priority hypotheses based on low-scoring areas
      assert.ok(markdownReport.includes('Priority Hypotheses to Test'));
      assert.ok(markdownReport.includes('1.'));
      assert.ok(markdownReport.includes('2.'));
    });

    test('saves structured report to projects directory', () => {
      const session = validator.createSession();
      const areas = validator.getAssessmentAreas();
      
      areas.forEach((area, index) => {
        validator.recordScore(session, area, index % 3);
      });
      
      const projectName = 'test-project';
      const filePath = validator.saveStructuredReport(session, projectName);
      
      // Verify file was created
      assert.ok(fs.existsSync(filePath));
      
      // Verify file is in correct location
      assert.ok(filePath.includes('/projects/'));
      assert.ok(filePath.includes('business-model-validation.md'));
      
      // Verify file content
      const fileContent = fs.readFileSync(filePath, 'utf8');
      assert.ok(fileContent.includes('BUSINESS MODEL VALIDATION REPORT'));
      
      // Clean up
      fs.unlinkSync(filePath);
      const projectDir = path.dirname(filePath);
      if (fs.existsSync(projectDir) && fs.readdirSync(projectDir).length === 0) {
        fs.rmdirSync(projectDir);
      }
    });

    test('creates project directory if it does not exist', () => {
      const session = validator.createSession();
      const areas = validator.getAssessmentAreas();
      
      areas.forEach((area, index) => {
        validator.recordScore(session, area, index % 3);
      });
      
      const projectName = 'new-test-project';
      const filePath = validator.saveStructuredReport(session, projectName);
      
      // Verify directory and file were created
      const projectDir = path.dirname(filePath);
      assert.ok(fs.existsSync(projectDir));
      assert.ok(fs.existsSync(filePath));
      
      // Clean up
      fs.unlinkSync(filePath);
      fs.rmdirSync(projectDir);
    });
  });
});