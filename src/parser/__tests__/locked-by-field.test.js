import { test, describe } from 'node:test';
import assert from 'node:assert';
import { parseFrontmatter, findNextAssertion } from '../index.js';

describe('Locked-By Field Tests', () => {
  test('parser reads locked-by field from YAML frontmatter', () => {
    const yamlContent = `---
id: test-assertion
parent: test-spec
created: 2026-02-25T19:30:00Z
priority: 1
status: in_progress
locked-by: builder-macbook-12345-1706210400
---

# Test Assertion`;

    const { data } = parseFrontmatter(yamlContent);

    assert.ok(data.lockedBy, 'Should parse locked-by field');
    assert.equal(data.lockedBy, 'builder-macbook-12345-1706210400', 'Should convert locked-by to lockedBy in camelCase');
  });

  test('parser handles assertions without locked-by field', () => {
    const yamlContent = `---
id: test-assertion
parent: test-spec
created: 2026-02-25T19:30:00Z
priority: 1
status: not_started
---

# Test Assertion`;

    const { data } = parseFrontmatter(yamlContent);

    assert.equal(data.lockedBy, undefined, 'Should not have lockedBy field');
  });

  test('spekk next skips assertions with status in_progress AND locked-by set', () => {
    const currentTime = Date.now();
    const recentTimestamp = Math.floor(currentTime / 1000) - 3600; // 1 hour ago

    const assertions = [
      {
        id: 'locked-assertion',
        parent: 'test-spec',
        priority: 1,
        status: 'in_progress',
        created: '2026-02-25T19:00:00Z',
        branch: 'main',
        lockedBy: `builder-test-12345-${recentTimestamp}`
      },
      {
        id: 'unlocked-assertion',
        parent: 'test-spec',
        priority: 2,
        status: 'not_started',
        created: '2026-02-25T19:01:00Z',
        branch: 'main'
      }
    ];

    const nextAssertion = findNextAssertion(assertions, [], { allBranches: true });

    assert.ok(nextAssertion, 'Should find an assertion');
    assert.equal(nextAssertion.id, 'unlocked-assertion', 'Should skip locked in_progress assertion');
  });

  test('spekk next includes assertions with status in_progress but no locked-by', () => {
    const assertions = [
      {
        id: 'in-progress-no-lock',
        parent: 'test-spec',
        priority: 1,
        status: 'in_progress',
        created: '2026-02-25T19:00:00Z',
        branch: 'main'
      },
      {
        id: 'not-started',
        parent: 'test-spec',
        priority: 2,
        status: 'not_started',
        created: '2026-02-25T19:01:00Z',
        branch: 'main'
      }
    ];

    const nextAssertion = findNextAssertion(assertions, [], { allBranches: true });

    assert.ok(nextAssertion, 'Should find an assertion');
    assert.equal(nextAssertion.id, 'in-progress-no-lock', 'Should include in_progress assertion without locked-by');
  });

  test('stale locks (>2 hours) are ignored by spekk next', () => {
    const currentTime = Date.now();
    const staleTimestamp = Math.floor(currentTime / 1000) - 7201; // 2 hours and 1 second ago

    const assertions = [
      {
        id: 'stale-locked-assertion',
        parent: 'test-spec',
        priority: 1,
        status: 'in_progress',
        created: '2026-02-25T19:00:00Z',
        branch: 'main',
        lockedBy: `builder-test-12345-${staleTimestamp}`
      },
      {
        id: 'other-assertion',
        parent: 'test-spec',
        priority: 2,
        status: 'not_started',
        created: '2026-02-25T19:01:00Z',
        branch: 'main'
      }
    ];

    const nextAssertion = findNextAssertion(assertions, [], { allBranches: true });

    assert.ok(nextAssertion, 'Should find an assertion');
    assert.equal(nextAssertion.id, 'stale-locked-assertion', 'Should include stale locked assertion since lock is expired');
  });

  test('fresh locks (<2 hours) prevent assertion selection', () => {
    const currentTime = Date.now();
    const freshTimestamp = Math.floor(currentTime / 1000) - 3600; // 1 hour ago

    const assertions = [
      {
        id: 'fresh-locked-assertion',
        parent: 'test-spec',
        priority: 1,
        status: 'in_progress',
        created: '2026-02-25T19:00:00Z',
        branch: 'main',
        lockedBy: `builder-test-12345-${freshTimestamp}`
      }
    ];

    const nextAssertion = findNextAssertion(assertions, [], { allBranches: true });

    assert.equal(nextAssertion, null, 'Should not find any assertion when only assertion has fresh lock');
  });

  test('lock format validation - extracts timestamp correctly', () => {
    const timestamp = 1706210400;
    const lockId = `builder-macbook-12345-${timestamp}`;

    const yamlContent = `---
id: test-assertion
parent: test-spec
created: 2026-02-25T19:30:00Z
priority: 1
status: in_progress
locked-by: ${lockId}
---

# Test Assertion`;

    const { data } = parseFrontmatter(yamlContent);

    assert.equal(data.lockedBy, lockId, 'Should parse full lock ID');

    // Verify timestamp can be extracted
    const extractedTimestamp = parseInt(lockId.split('-').pop());
    assert.equal(extractedTimestamp, timestamp, 'Should be able to extract timestamp from lock ID');
  });

  test('locked-by field does not affect done assertions', () => {
    const currentTime = Date.now();
    const freshTimestamp = Math.floor(currentTime / 1000) - 3600;

    const assertions = [
      {
        id: 'done-locked',
        parent: 'test-spec',
        priority: 1,
        status: 'done',
        created: '2026-02-25T19:00:00Z',
        branch: 'main',
        lockedBy: `builder-test-12345-${freshTimestamp}`
      },
      {
        id: 'not-started',
        parent: 'test-spec',
        priority: 2,
        status: 'not_started',
        created: '2026-02-25T19:02:00Z',
        branch: 'main'
      }
    ];

    const nextAssertion = findNextAssertion(assertions, [], { allBranches: true });

    // Done assertions should be filtered out before lock checking
    assert.equal(nextAssertion.id, 'not-started', 'Should skip done regardless of lock');
  });

  test('failed assertions without lock are included in work queue', () => {
    const assertions = [
      {
        id: 'failed-no-lock',
        parent: 'test-spec',
        priority: 1,
        status: 'failed',
        created: '2026-02-25T19:00:00Z',
        branch: 'main'
      },
      {
        id: 'not-started',
        parent: 'test-spec',
        priority: 2,
        status: 'not_started',
        created: '2026-02-25T19:01:00Z',
        branch: 'main'
      }
    ];

    const nextAssertion = findNextAssertion(assertions, [], { allBranches: true });

    // Failed assertions need work, so they should be included
    assert.equal(nextAssertion.id, 'failed-no-lock', 'Should include failed assertion without lock');
  });
});
