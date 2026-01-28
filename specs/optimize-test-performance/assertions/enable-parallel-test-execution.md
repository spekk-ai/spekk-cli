---
id: enable-parallel-test-execution
parent: optimize-test-performance
created: 2026-01-28T21:25:00Z
priority: 2
status: done
---

# Enable Parallel Test Execution

## What Must Be True

Tests run in parallel instead of sequentially to improve performance.

## Current Issue

Test runner uses `--test-concurrency=1` which forces sequential execution:
```javascript
spawn('node', ['--test', '--test-concurrency=1', ...testFiles])
```

## Success Criteria

- ✅ Remove `--test-concurrency=1` flag to enable parallel execution
- ✅ Tests are isolated and don't interfere with each other
- ✅ No shared state between tests
- ✅ Proper cleanup after each test

## Implementation

**Update test runner:**
```javascript
// From: ['--test', '--test-concurrency=1', ...testFiles]
// To:   ['--test', ...testFiles]
```

**Ensure test isolation:**
- No shared global variables
- Proper mocking and cleanup
- Independent file system operations (if any)

## Validation

Tests should still pass when run in parallel, but complete faster.