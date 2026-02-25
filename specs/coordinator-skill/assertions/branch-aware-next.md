---
id: branch-aware-next
parent: coordinator-skill
created: 2026-02-25T00:00:00Z
priority: 1
status: not_started
depends-on: parser-depends-on-support
---

# spekk next is Branch-Aware by Default

The `spekk next` command filters to current git branch and respects dependencies when selecting the next assertion.

## Success Criteria

### Default Behavior (Branch-Aware)

```bash
$ git branch
* feature/chat-system
  main

$ spekk next
{
  "type": "assertion",
  "id": "chat-session-model",
  "parent": "chat-system",
  "branch": "feature/chat-system",
  "dependsOn": "websocket-connection",
  "status": "not_started",
  ...
}
```

**Selection logic:**
1. Filter to assertions where `branch` matches current git branch
2. Filter to assertions where `status` ≠ `done`
3. Filter to assertions where dependencies are satisfied:
   - `dependsOn` is null/omitted, OR
   - Assertion referenced by `dependsOn` has `status: done`
4. Sort by priority (lower = higher priority)
5. Break ties by creation timestamp (older = higher priority)
6. Return first result

### Dependency Blocking

```bash
# websocket-connection is not_started
# chat-session-model depends-on: websocket-connection

$ spekk next
{
  "id": "websocket-connection",  # Returns this (no dep)
  ...
}

# After completing websocket-connection
$ spekk next
{
  "id": "chat-session-model",  # Now unblocked
  "dependsOn": "websocket-connection",
  ...
}
```

### Global Mode (Backward Compatibility)

```bash
$ spekk next --all-branches
# Ignores current branch, returns next assertion globally
# Useful for debugging or testing
```

### No Git Repository

If not in a git repository or can't detect current branch:
```bash
$ spekk next
Warning: Not in a git repository. Defaulting to assertions with branch='main'.
{
  "id": "some-assertion",
  "branch": "main",
  ...
}
```

### No Assertions Ready

If all assertions on current branch are done or blocked:
```bash
$ spekk next
No assertions ready on branch 'feature/chat-system'.

Status:
- 2 done
- 1 blocked (waiting for websocket-connection)

Try:
  git checkout main
  spekk next
```

## Implementation

### Get Current Git Branch

```javascript
import { execSync } from 'child_process';

function getCurrentGitBranch() {
  try {
    const branch = execSync('git branch --show-current', { 
      encoding: 'utf8',
      stdio: ['pipe', 'pipe', 'ignore']  // Suppress stderr
    }).trim();
    
    if (!branch) {
      console.warn('Warning: Not in a git repository. Defaulting to main.');
      return 'main';
    }
    
    return branch;
  } catch (error) {
    console.warn('Warning: Could not detect git branch. Defaulting to main.');
    return 'main';
  }
}
```

### Filter and Sort Logic

```javascript
function getNextAssertion(options = {}) {
  const allAssertions = parseAllAssertions();
  const currentBranch = options.allBranches ? null : getCurrentGitBranch();
  
  // Filter to current branch (unless --all-branches)
  let assertions = currentBranch
    ? allAssertions.filter(a => a.branch === currentBranch)
    : allAssertions;
  
  // Filter out done assertions
  assertions = assertions.filter(a => a.status !== 'done');
  
  // Filter by dependency satisfaction
  assertions = assertions.filter(a => {
    if (!a.dependsOn) return true;  // No dependency
    
    const dependency = allAssertions.find(d => d.id === a.dependsOn);
    return dependency && dependency.status === 'done';
  });
  
  // Sort by priority (ascending), then by created (ascending)
  assertions.sort((a, b) => {
    if (a.priority !== b.priority) {
      return a.priority - b.priority;  // Lower priority number = higher priority
    }
    return new Date(a.created) - new Date(b.created);  // Older = higher priority
  });
  
  return assertions[0] || null;
}
```

### CLI Integration

Update `src/parser/cli.js`:

```javascript
if (args.includes('--all-branches')) {
  const next = getNextAssertion({ allBranches: true });
  // ...
} else {
  const next = getNextAssertion();  // Branch-aware by default
  // ...
}
```

### User-Friendly Output

```bash
$ spekk next
Next assertion: chat-session-model (feature/chat-system)
Priority: 1
Status: not_started
Depends on: websocket-connection (✅ done)

File: specs/chat-system/assertions/chat-session-model.md
```

## Validation

- `spekk next` detects current git branch
- Returns assertions matching current branch
- Filters out done assertions
- Filters out blocked assertions (dependencies not done)
- Sorts by priority, then creation timestamp
- `--all-branches` flag disables branch filtering
- Gracefully handles missing git repository
- Clear messages when no assertions are ready

**Tests:**
- `src/parser/__tests__/branch-aware-next.test.js`
- Test branch filtering
- Test dependency blocking
- Test sorting (priority + timestamp)
- Test --all-branches flag
- Test no-git-repo scenario
- Test no-ready-assertions scenario
