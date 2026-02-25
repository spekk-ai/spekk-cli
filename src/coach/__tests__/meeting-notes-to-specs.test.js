import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import { MeetingNotesToSpecs } from '../meeting-notes-to-specs.js';

describe('MeetingNotesToSpecs Skill', () => {
  const skill = new MeetingNotesToSpecs();

  describe('Skill Interface', () => {
    test('should return correct ID', () => {
      assert.strictEqual(skill.getId(), 'meeting-notes-to-specs');
    });

    test('should return correct name', () => {
      assert.strictEqual(skill.getName(), 'Meeting Notes to Specs');
    });

    test('should return a description', () => {
      const desc = skill.getDescription();
      assert.ok(desc.length > 0);
      assert.ok(desc.includes('meeting'));
    });

    test('should return questions', () => {
      const questions = skill.getQuestions();
      assert.ok(Array.isArray(questions));
      assert.ok(questions.length > 0);
      assert.strictEqual(questions[0].id, 'transcript');
    });

    test('should process responses', () => {
      const result = skill.processResponses({ transcript: 'Some meeting notes' });
      assert.ok(result.summary);
      assert.ok(Array.isArray(result.recommendations));
      assert.ok(result.data);
    });

    test('should return output format', () => {
      const format = skill.getOutputFormat();
      assert.strictEqual(format.format, 'markdown');
      assert.ok(format.sections.includes('specs'));
    });
  });

  describe('Trigger Detection', () => {
    test('should trigger on meeting-related keywords', () => {
      assert.ok(skill.shouldTrigger('Here are my meeting notes'));
      assert.ok(skill.shouldTrigger('Process meeting transcript'));
      assert.ok(skill.shouldTrigger('meeting summary from today'));
      assert.ok(skill.shouldTrigger('from our meeting yesterday'));
      assert.ok(skill.shouldTrigger('standup notes'));
      assert.ok(skill.shouldTrigger('retro notes from sprint'));
      assert.ok(skill.shouldTrigger('planning notes'));
    });

    test('should not trigger on unrelated input', () => {
      assert.ok(!skill.shouldTrigger('add dark mode'));
      assert.ok(!skill.shouldTrigger('fix the login bug'));
      assert.ok(!skill.shouldTrigger('validate business model'));
    });

    test('should be case insensitive', () => {
      assert.ok(skill.shouldTrigger('MEETING NOTES'));
      assert.ok(skill.shouldTrigger('Meeting Transcript'));
    });
  });

  describe('Kebab Case Conversion', () => {
    test('should convert spaces to kebab-case', () => {
      assert.strictEqual(skill.toKebabCase('dark mode'), 'dark-mode');
    });

    test('should convert camelCase to kebab-case', () => {
      assert.strictEqual(skill.toKebabCase('darkMode'), 'dark-mode');
    });

    test('should handle already kebab-case strings', () => {
      assert.strictEqual(skill.toKebabCase('dark-mode'), 'dark-mode');
    });

    test('should remove special characters', () => {
      assert.strictEqual(skill.toKebabCase('dark mode!@#'), 'dark-mode');
    });

    test('should handle underscores', () => {
      assert.strictEqual(skill.toKebabCase('dark_mode_toggle'), 'dark-mode-toggle');
    });

    test('should collapse multiple dashes', () => {
      assert.strictEqual(skill.toKebabCase('dark--mode'), 'dark-mode');
    });

    test('should trim leading/trailing dashes', () => {
      assert.strictEqual(skill.toKebabCase('-dark-mode-'), 'dark-mode');
    });
  });

  describe('Priority Validation', () => {
    test('should accept valid priorities', () => {
      assert.strictEqual(skill.validatePriority(1), 1);
      assert.strictEqual(skill.validatePriority(2), 2);
      assert.strictEqual(skill.validatePriority(3), 3);
    });

    test('should clamp to range 1-3', () => {
      assert.strictEqual(skill.validatePriority(0), 1);
      assert.strictEqual(skill.validatePriority(-1), 1);
      assert.strictEqual(skill.validatePriority(5), 3);
      assert.strictEqual(skill.validatePriority(100), 3);
    });

    test('should handle string inputs', () => {
      assert.strictEqual(skill.validatePriority('2'), 2);
    });

    test('should default NaN to 1', () => {
      assert.strictEqual(skill.validatePriority('abc'), 1);
    });
  });

  describe('Parent Spec Generation', () => {
    test('should generate valid YAML frontmatter', () => {
      const content = skill.generateParentSpec({
        id: 'dark-mode',
        title: 'Dark Mode',
        description: 'Add dark mode support.',
        created: '2026-01-20T17:00:00Z',
        priority: 2,
        successCriteria: ['User can toggle dark mode']
      });

      assert.ok(content.startsWith('---\n'));
      assert.ok(content.includes('id: dark-mode'));
      assert.ok(content.includes('created: 2026-01-20T17:00:00Z'));
      assert.ok(content.includes('priority: 2'));
      // Parent specs should NOT have status field
      assert.ok(!content.includes('status:'));
      assert.ok(content.includes('# Dark Mode'));
      assert.ok(content.includes('Add dark mode support.'));
      assert.ok(content.includes('## Success Criteria'));
      assert.ok(content.includes('- User can toggle dark mode'));
    });

    test('should not include status field in parent spec', () => {
      const content = skill.generateParentSpec({
        id: 'test-spec',
        title: 'Test Spec',
        description: '',
        created: '2026-01-20T17:00:00Z',
        priority: 1,
        successCriteria: []
      });

      assert.ok(!content.includes('status'));
    });
  });

  describe('Assertion Generation', () => {
    test('should generate assertion with correct YAML frontmatter', () => {
      const result = skill.generateAssertion({
        id: 'theme-toggle',
        parent: 'dark-mode',
        title: 'Theme Toggle in Settings',
        description: 'User can toggle between light and dark themes.',
        created: '2026-01-20T17:00:00Z',
        priority: 1,
        successCriteria: ['Toggle switch exists in settings', 'Theme changes immediately on toggle']
      });

      assert.strictEqual(result.id, 'theme-toggle');
      assert.strictEqual(result.title, 'Theme Toggle in Settings');
      assert.ok(result.content.includes('id: theme-toggle'));
      assert.ok(result.content.includes('parent: dark-mode'));
      assert.ok(result.content.includes('created: 2026-01-20T17:00:00Z'));
      assert.ok(result.content.includes('priority: 1'));
      assert.ok(result.content.includes('status: not_started'));
      assert.ok(result.content.includes('# Theme Toggle in Settings'));
      assert.ok(result.content.includes('## Success Criteria'));
      assert.ok(result.content.includes('- Toggle switch exists in settings'));
    });

    test('should always set status to not_started', () => {
      const result = skill.generateAssertion({
        id: 'test-assertion',
        parent: 'test-spec',
        title: 'Test Assertion',
        description: '',
        created: '2026-01-20T17:00:00Z',
        priority: 2,
        successCriteria: []
      });

      assert.ok(result.content.includes('status: not_started'));
    });
  });

  describe('Feature to Spec Conversion', () => {
    test('should convert a feature to a complete spec structure', () => {
      const spec = skill.featureToSpec({
        id: 'dark-mode',
        title: 'Dark Mode',
        description: 'Add dark mode support to the app.',
        priority: 2,
        created: '2026-01-20T17:00:00Z',
        successCriteria: ['Users can switch between light and dark themes'],
        assertions: [
          {
            id: 'theme-toggle',
            title: 'Theme Toggle in Settings',
            description: 'User can toggle theme.',
            priority: 1,
            successCriteria: ['Toggle switch in settings page']
          },
          {
            id: 'preference-persisted',
            title: 'Preference Persisted',
            description: 'Theme choice saved across sessions.',
            priority: 1,
            successCriteria: ['Theme persists after page reload']
          }
        ]
      });

      assert.strictEqual(spec.specId, 'dark-mode');
      assert.strictEqual(spec.directory, 'specs/dark-mode');
      assert.ok(spec.parentSpec.includes('id: dark-mode'));
      assert.strictEqual(spec.assertions.length, 2);
      assert.strictEqual(spec.assertions[0].id, 'theme-toggle');
      assert.strictEqual(spec.assertions[1].id, 'preference-persisted');

      // Check file paths
      assert.strictEqual(spec.files.length, 3); // 1 parent + 2 assertions
      assert.strictEqual(spec.files[0].path, 'specs/dark-mode/dark-mode.md');
      assert.strictEqual(spec.files[1].path, 'specs/dark-mode/assertions/theme-toggle.md');
      assert.strictEqual(spec.files[2].path, 'specs/dark-mode/assertions/preference-persisted.md');
    });

    test('should enforce kebab-case IDs', () => {
      const spec = skill.featureToSpec({
        id: 'Dark Mode Feature',
        title: 'Dark Mode',
        assertions: [{
          id: 'Theme Toggle',
          title: 'Theme Toggle',
          successCriteria: ['Works']
        }]
      });

      assert.strictEqual(spec.specId, 'dark-mode-feature');
      assert.strictEqual(spec.assertions[0].id, 'theme-toggle');
    });

    test('should throw on missing id', () => {
      assert.throws(
        () => skill.featureToSpec({ title: 'No ID' }),
        /Feature must have id and title/
      );
    });

    test('should throw on missing title', () => {
      assert.throws(
        () => skill.featureToSpec({ id: 'no-title' }),
        /Feature must have id and title/
      );
    });

    test('should default priority to 2', () => {
      const spec = skill.featureToSpec({
        id: 'test',
        title: 'Test',
        created: '2026-01-20T17:00:00Z',
        assertions: []
      });

      assert.ok(spec.parentSpec.includes('priority: 2'));
    });
  });

  describe('Multiple Features to Multiple Specs', () => {
    test('should create separate specs for each feature', () => {
      const specs = skill.featuresToSpecs([
        {
          id: 'dark-mode',
          title: 'Dark Mode',
          priority: 2,
          assertions: [
            { id: 'toggle', title: 'Toggle', successCriteria: ['Toggle works'] }
          ]
        },
        {
          id: 'notifications',
          title: 'Push Notifications',
          priority: 1,
          assertions: [
            { id: 'subscribe', title: 'Subscribe', successCriteria: ['Can subscribe'] },
            { id: 'receive', title: 'Receive', successCriteria: ['Gets notifications'] }
          ]
        }
      ]);

      assert.strictEqual(specs.length, 2);
      assert.strictEqual(specs[0].specId, 'dark-mode');
      assert.strictEqual(specs[1].specId, 'notifications');

      // Each spec has independent directory
      assert.strictEqual(specs[0].directory, 'specs/dark-mode');
      assert.strictEqual(specs[1].directory, 'specs/notifications');

      // Each spec has correct assertion count
      assert.strictEqual(specs[0].assertions.length, 1);
      assert.strictEqual(specs[1].assertions.length, 2);
    });

    test('should throw on empty features array', () => {
      assert.throws(
        () => skill.featuresToSpecs([]),
        /Features must be a non-empty array/
      );
    });

    test('should throw on non-array input', () => {
      assert.throws(
        () => skill.featuresToSpecs('not an array'),
        /Features must be a non-empty array/
      );
    });
  });

  describe('Proposal Formatting', () => {
    test('should format a readable proposal', () => {
      const specs = skill.featuresToSpecs([
        {
          id: 'dark-mode',
          title: 'Dark Mode',
          priority: 2,
          created: '2026-01-20T17:00:00Z',
          successCriteria: ['Theme switching works'],
          assertions: [
            {
              id: 'theme-toggle',
              title: 'Theme Toggle',
              priority: 1,
              successCriteria: ['Toggle in settings']
            }
          ]
        }
      ]);

      const proposal = skill.formatProposal(specs);

      assert.ok(proposal.includes('Proposed Specs from Meeting'));
      assert.ok(proposal.includes('Spec 1: Dark Mode'));
      assert.ok(proposal.includes('Priority: 2'));
      assert.ok(proposal.includes('Theme Toggle'));
      assert.ok(proposal.includes('priority 1'));
      assert.ok(proposal.includes('Shall I create these spec files?'));
    });

    test('should format multiple specs in proposal', () => {
      const specs = skill.featuresToSpecs([
        {
          id: 'feature-a',
          title: 'Feature A',
          priority: 1,
          created: '2026-01-20T17:00:00Z',
          assertions: [{ id: 'a1', title: 'A1', successCriteria: ['Done'] }]
        },
        {
          id: 'feature-b',
          title: 'Feature B',
          priority: 3,
          created: '2026-01-20T17:00:00Z',
          assertions: [{ id: 'b1', title: 'B1', successCriteria: ['Done'] }]
        }
      ]);

      const proposal = skill.formatProposal(specs);
      assert.ok(proposal.includes('Spec 1: Feature A'));
      assert.ok(proposal.includes('Spec 2: Feature B'));
    });
  });

  describe('File Writing', () => {
    test('should write spec files to correct directory structure', () => {
      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-'));

      try {
        const spec = skill.featureToSpec({
          id: 'test-feature',
          title: 'Test Feature',
          description: 'A test feature.',
          created: '2026-01-20T17:00:00Z',
          priority: 1,
          successCriteria: ['It works'],
          assertions: [
            {
              id: 'test-assertion',
              title: 'Test Assertion',
              priority: 1,
              successCriteria: ['Passes tests']
            }
          ]
        });

        const createdFiles = skill.writeSpecFiles(spec, tmpDir);

        // Verify files were created
        assert.strictEqual(createdFiles.length, 2);
        assert.ok(createdFiles.includes('specs/test-feature/test-feature.md'));
        assert.ok(createdFiles.includes('specs/test-feature/assertions/test-assertion.md'));

        // Verify directory structure
        assert.ok(fs.existsSync(path.join(tmpDir, 'specs/test-feature')));
        assert.ok(fs.existsSync(path.join(tmpDir, 'specs/test-feature/assertions')));

        // Verify file contents
        const parentContent = fs.readFileSync(
          path.join(tmpDir, 'specs/test-feature/test-feature.md'), 'utf8'
        );
        assert.ok(parentContent.includes('id: test-feature'));
        assert.ok(parentContent.includes('priority: 1'));
        assert.ok(!parentContent.includes('status:'));

        const assertionContent = fs.readFileSync(
          path.join(tmpDir, 'specs/test-feature/assertions/test-assertion.md'), 'utf8'
        );
        assert.ok(assertionContent.includes('id: test-assertion'));
        assert.ok(assertionContent.includes('parent: test-feature'));
        assert.ok(assertionContent.includes('status: not_started'));
      } finally {
        // Clean up
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });
  });
});
