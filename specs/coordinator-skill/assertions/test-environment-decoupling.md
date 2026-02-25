---
id: test-environment-decoupling
parent: coordinator-skill
created: 2026-02-25T11:00:00Z
priority: 1
status: done
---

# Tests Are Decoupled From Environment

All tests pass regardless of current git branch or environment state.

## Problem

**branch-aware-next.test.js fails** because it expects fixtures matching the current git branch:
```javascript
const currentBranch = execSync('git branch --show-current', { encoding: 'utf8' }).trim();
// Test expects fixture for currentBranch, but only has feature/chat-system and main
```

**parser-basic.test.js fails** because existing "Next Priority Identification" tests don't account for branch-aware filtering introduced by branch-aware-next assertion.

## Success Criteria

- ✅ All tests pass on any git branch
- ✅ Tests use mocked git branch instead of actual environment
- ✅ Parser-basic tests updated for branch-aware behavior
- ✅ No flaky tests due to environment coupling

## Implementation

### Fix branch-aware-next.test.js

Mock `getCurrentGitBranch()` to return controlled values:

```javascript
import { test } from 'node:test';
import * as parserModule from '../index.js';

test('returns assertion matching mocked branch', (t) => {
  // Mock the git branch function
  t.mock.method(parserModule, 'getCurrentGitBranch', () => 'feature/chat-system');
  
  run({ specsDirectory: 'src/parser/__tests__/fixtures/branch-aware' });
  
  const output = JSON.parse(logOutput[0]);
  assert.strictEqual(output.branch, 'feature/chat-system');
});
```

### Fix parser-basic.test.js

Update "Next Priority Identification" tests to account for branch filtering:
- Either add `--all-branches` flag to test invocations
- Or mock `getCurrentGitBranch()` to return 'main'
- Or update test fixtures to have assertions on current branch

## Validation

```bash
# Should pass on any branch
git checkout main
npm test  # ✅ All tests pass

git checkout feature/coordinator-skill  
npm test  # ✅ All tests pass

git checkout -b random-branch
npm test  # ✅ All tests pass
```

**Tests:** src/parser/__tests__/branch-aware-next.test.js (fix), src/parser/__tests__/parser-basic.test.js (fix)
