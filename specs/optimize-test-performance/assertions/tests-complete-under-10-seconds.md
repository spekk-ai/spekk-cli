---
id: tests-complete-under-10-seconds
parent: optimize-test-performance
created: 2026-01-28T21:25:00Z
priority: 1
status: not_started
---

# Tests Complete Under 10 Seconds

## What Must Be True

The entire test suite completes in under 10 seconds, with individual tests completing in under 100ms.

## Current Performance Issues

**Slow tests identified:**
- `Orchestration Loops` - Over 10 seconds (spawn real processes)
- `Coach CLI` - Over 1 second (file system operations)
- Various CLI tests - File system heavy

## Success Criteria

- ✅ `npm test` completes in < 10 seconds total
- ✅ Individual tests complete in < 100ms
- ✅ No test timeouts in CI
- ✅ Tests use mocks instead of real processes/file system

## Optimization Strategies

1. **Mock external dependencies** - Don't spawn real processes
2. **Use in-memory file system** - For tests that need file operations
3. **Minimize I/O operations** - Use stubs and fixtures
4. **Parallel test execution** - Remove `--test-concurrency=1` if safe
5. **Remove unnecessary setup/teardown** - Only test what's needed

## Validation

```bash
time npm test  # Should complete in < 10 seconds
```