import { test, describe } from 'node:test';
import assert from 'node:assert';
import { MeetingNotesToSpecs } from '../meeting-notes-to-specs.js';

describe('Extract Three Categories from Transcript', () => {
  const skill = new MeetingNotesToSpecs();

  describe('extractCategories', () => {
    test('should separate meeting data into three distinct categories', () => {
      const result = skill.extractCategories({
        todos: [
          { description: 'Send spreadsheet to team', owner: 'marcy' }
        ],
        features: [
          { id: 'dark-mode', title: 'Dark Mode', description: 'Add dark mode support' }
        ],
        decisions: [
          { decision: 'Use PostgreSQL for the database', context: 'Better JSON support' }
        ]
      });

      assert.ok(result.todos, 'Should have todos category');
      assert.ok(result.features, 'Should have features category');
      assert.ok(result.decisions, 'Should have decisions category');
      assert.strictEqual(result.todos.length, 1);
      assert.strictEqual(result.features.length, 1);
      assert.strictEqual(result.decisions.length, 1);
    });

    test('should return three separate arrays (todos ≠ features ≠ decisions)', () => {
      const result = skill.extractCategories({
        todos: [
          { description: 'Follow up on design', owner: 'alice' },
          { description: 'Review PRs', owner: 'bob' }
        ],
        features: [
          { id: 'notifications', title: 'Push Notifications', description: 'Add push notifications' }
        ],
        decisions: [
          { decision: 'Use React for frontend' },
          { decision: 'Deploy to AWS', context: 'Team has experience' }
        ]
      });

      // Each category is a separate array
      assert.ok(Array.isArray(result.todos));
      assert.ok(Array.isArray(result.features));
      assert.ok(Array.isArray(result.decisions));

      // Counts are independent
      assert.strictEqual(result.todos.length, 2);
      assert.strictEqual(result.features.length, 1);
      assert.strictEqual(result.decisions.length, 2);
    });

    test('should handle empty categories gracefully', () => {
      const result = skill.extractCategories({
        todos: [],
        features: [],
        decisions: []
      });

      assert.deepStrictEqual(result.todos, []);
      assert.deepStrictEqual(result.features, []);
      assert.deepStrictEqual(result.decisions, []);
    });

    test('should default missing categories to empty arrays', () => {
      const result = skill.extractCategories({});

      assert.deepStrictEqual(result.todos, []);
      assert.deepStrictEqual(result.features, []);
      assert.deepStrictEqual(result.decisions, []);
    });

    test('should include a summary with counts', () => {
      const result = skill.extractCategories({
        todos: [{ description: 'Task 1', owner: 'alice' }],
        features: [
          { id: 'feat-a', title: 'Feature A', description: 'Desc A' },
          { id: 'feat-b', title: 'Feature B', description: 'Desc B' }
        ],
        decisions: [
          { decision: 'Decision 1' },
          { decision: 'Decision 2' },
          { decision: 'Decision 3' }
        ]
      });

      assert.ok(result.summary, 'Should include summary');
      assert.strictEqual(result.summary.todoCount, 1);
      assert.strictEqual(result.summary.featureCount, 2);
      assert.strictEqual(result.summary.decisionCount, 3);
      assert.strictEqual(result.summary.totalItems, 6);
    });

    test('should preserve todo item structure', () => {
      const result = skill.extractCategories({
        todos: [
          { description: 'Send report', owner: 'marcy', dueDate: '2026-03-01' }
        ],
        features: [],
        decisions: []
      });

      assert.strictEqual(result.todos[0].description, 'Send report');
      assert.strictEqual(result.todos[0].owner, 'marcy');
    });

    test('should preserve feature item structure', () => {
      const result = skill.extractCategories({
        todos: [],
        features: [
          { id: 'dark-mode', title: 'Dark Mode', description: 'Theme support', priority: 2 }
        ],
        decisions: []
      });

      assert.strictEqual(result.features[0].id, 'dark-mode');
      assert.strictEqual(result.features[0].title, 'Dark Mode');
      assert.strictEqual(result.features[0].description, 'Theme support');
    });

    test('should preserve decision item structure', () => {
      const result = skill.extractCategories({
        todos: [],
        features: [],
        decisions: [
          { decision: 'Use PostgreSQL', context: 'Better JSON support' }
        ]
      });

      assert.strictEqual(result.decisions[0].decision, 'Use PostgreSQL');
      assert.strictEqual(result.decisions[0].context, 'Better JSON support');
    });
  });

  describe('Category validation', () => {
    test('should reject non-object input', () => {
      assert.throws(
        () => skill.extractCategories(null),
        /meetingData must be an object/
      );
      assert.throws(
        () => skill.extractCategories('string'),
        /meetingData must be an object/
      );
    });

    test('should reject non-array category values', () => {
      assert.throws(
        () => skill.extractCategories({ todos: 'not an array', features: [], decisions: [] }),
        /todos must be an array/
      );
      assert.throws(
        () => skill.extractCategories({ todos: [], features: 'bad', decisions: [] }),
        /features must be an array/
      );
      assert.throws(
        () => skill.extractCategories({ todos: [], features: [], decisions: 42 }),
        /decisions must be an array/
      );
    });
  });

  describe('processResponses returns three-category structure', () => {
    test('should include outputTypes with all three categories', () => {
      const result = skill.processResponses({ transcript: 'Some meeting notes' });
      assert.ok(result.data.outputTypes.includes('todos'));
      assert.ok(result.data.outputTypes.includes('specs'));
      assert.ok(result.data.outputTypes.includes('context'));
    });
  });

  describe('Integration: extractCategories feeds into existing methods', () => {
    test('features from extractCategories can be passed to featuresToSpecs', () => {
      const categories = skill.extractCategories({
        todos: [],
        features: [
          {
            id: 'dark-mode',
            title: 'Dark Mode',
            description: 'Add dark mode support',
            priority: 2,
            assertions: [
              { id: 'toggle', title: 'Toggle Switch', successCriteria: ['Toggle works'] }
            ]
          }
        ],
        decisions: []
      });

      const specs = skill.featuresToSpecs(categories.features);
      assert.strictEqual(specs.length, 1);
      assert.strictEqual(specs[0].specId, 'dark-mode');
    });

    test('decisions from extractCategories can be passed to formatDecisions', () => {
      const categories = skill.extractCategories({
        todos: [],
        features: [],
        decisions: [
          { decision: 'Use PostgreSQL', context: 'Better JSON support' }
        ]
      });

      const formatted = skill.formatDecisions(categories.decisions, '2026-02-12');
      assert.ok(formatted.includes('Use PostgreSQL'));
      assert.ok(formatted.includes('2026-02-12'));
    });
  });
});
