import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const projectRoot = path.join(__dirname, '../../..');
const builderPromptPath = path.join(projectRoot, 'specs/builder-agent/builder.prompt.md');

describe('Builder Locks and Unlocks Assertions', () => {
  let promptContent;

  // Load the builder prompt once for all tests
  promptContent = fs.readFileSync(builderPromptPath, 'utf8');

  test('builder prompt instructs adding locked-by field when marking in_progress', () => {
    // The prompt must instruct builders to add locked-by when claiming work
    assert.ok(
      promptContent.includes('locked-by') && promptContent.includes('in_progress'),
      'Prompt must mention locked-by field in context of in_progress status'
    );
    // Specifically: "When marking `in_progress` → ADD `locked-by` field"
    assert.ok(
      promptContent.includes('When marking `in_progress`') && promptContent.includes('ADD `locked-by`'),
      'Prompt must explicitly instruct adding locked-by when marking in_progress'
    );
  });

  test('builder prompt specifies lock format as builder-{hostname}-{pid}-{timestamp}', () => {
    assert.ok(
      promptContent.includes('builder-{hostname}-{pid}-{timestamp}'),
      'Prompt must specify lock format: builder-{hostname}-{pid}-{timestamp}'
    );
    // Also check it gives real examples of how to generate each part
    assert.ok(
      promptContent.includes('hostname') && promptContent.includes('$$') && promptContent.includes('date +%s'),
      'Prompt must include instructions for generating hostname, PID ($$), and timestamp (date +%s)'
    );
  });

  test('builder prompt instructs committing lock immediately before starting work', () => {
    assert.ok(
      promptContent.includes('Commit lock changes immediately') ||
      promptContent.includes('commit lock changes immediately') ||
      promptContent.includes('Commit immediately'),
      'Prompt must instruct committing lock changes immediately before starting work'
    );
    // Check the critical lock rules section has this
    assert.ok(
      promptContent.includes('before starting work'),
      'Prompt must specify commit happens before starting implementation work'
    );
  });

  test('builder prompt instructs pulling after committing to detect conflicts', () => {
    assert.ok(
      promptContent.includes('Pull after committing lock to detect conflicts') ||
      promptContent.includes('pull after committing') ||
      promptContent.includes('Pull after committing'),
      'Prompt must instruct pulling after committing to detect conflicts'
    );
  });

  test('builder prompt instructs picking next assertion on conflict', () => {
    assert.ok(
      promptContent.includes('conflict') && promptContent.includes('pick next assertion'),
      'Prompt must instruct picking next assertion if conflict is detected'
    );
  });

  test('builder prompt instructs removing locked-by when marking done', () => {
    // "When marking `done` or `failed` → REMOVE `locked-by` field completely"
    assert.ok(
      promptContent.includes('REMOVE `locked-by` field completely'),
      'Prompt must instruct removing locked-by field completely when done/failed'
    );
    // Check the done status section shows no locked-by
    assert.ok(
      promptContent.includes('done') && promptContent.includes('Remove locked-by field completely'),
      'Prompt completion section must show locked-by removal'
    );
  });

  test('builder prompt instructs removing locked-by when marking failed', () => {
    // The prompt's critical lock rules must mention both done and failed
    assert.ok(
      promptContent.includes('done` or `failed') && promptContent.includes('REMOVE'),
      'Prompt must instruct removing locked-by for both done and failed statuses'
    );
    // Also check the status values section
    assert.ok(
      promptContent.includes('`failed` - Implementation has confirmed issues') &&
      promptContent.includes('(no lock)'),
      'Failed status description must indicate no lock'
    );
  });

  test('builder prompt includes complete locking instructions section', () => {
    // Verify section 5 exists with locking content
    assert.ok(
      promptContent.includes('### 5. Update Status and Release Lock'),
      'Prompt must have section 5 titled "Update Status and Release Lock"'
    );
    // Verify all critical lock rules are present
    const criticalRules = [
      'When marking `in_progress` → ADD `locked-by` field',
      'When marking `done` or `failed` → REMOVE `locked-by` field completely',
      'Commit lock changes immediately (before starting work)',
      'Pull after committing lock to detect conflicts',
      'If conflict (someone else claimed it), pick next assertion'
    ];
    for (const rule of criticalRules) {
      assert.ok(
        promptContent.includes(rule),
        `Prompt must include critical lock rule: "${rule}"`
      );
    }
  });

  test('lock ID format is valid and extractable', () => {
    // Validate the lock format pattern can parse real lock IDs
    const lockPattern = /^builder-[a-zA-Z0-9._-]+-\d+-\d+$/;

    const validLockIds = [
      'builder-macbook-pro-12345-1706210400',
      'builder-warespace-i7-119237-1772133248',
      'builder-ci-server-1-99999-1700000000',
    ];

    for (const lockId of validLockIds) {
      assert.ok(lockPattern.test(lockId), `Lock ID should match pattern: ${lockId}`);

      // Verify timestamp is extractable from the last segment
      const parts = lockId.split('-');
      const timestamp = parseInt(parts[parts.length - 1]);
      assert.ok(!isNaN(timestamp), `Should extract valid timestamp from lock ID: ${lockId}`);
      assert.ok(timestamp > 0, `Timestamp should be positive: ${timestamp}`);
    }
  });
});
