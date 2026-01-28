---
id: tests-complete-under-30-seconds
parent: optimize-test-performance
created: 2026-01-28T21:25:00Z
priority: 1
status: done
---

# Tests Complete Under 30 Seconds

## What Must Be True

The entire test suite completes in under 30 seconds, with individual tests completing in under 500ms.

## Current Performance Issues

**Slow tests identified:**
- `Orchestration Loops` - Over 10 seconds (spawn real processes)
- `Coach CLI` - Over 1 second (file system operations)
- Various CLI tests - File system heavy

## Success Criteria

- ✅ `npm test` completes in < 30 seconds total
- ✅ Individual tests complete in < 500ms
- ✅ No test timeouts in CI
- ✅ Tests use mocks instead of real processes/file system
- ✅ All 264 tests pass (currently 37 failing)

## Optimization Strategies

1. **Fix failing tests** - 37 tests currently failing
2. **Mock external dependencies** - Don't spawn real processes
3. **Use in-memory file system** - For tests that need file operations
4. **Minimize I/O operations** - Use stubs and fixtures
5. **Parallel test execution** - Remove `--test-concurrency=1` if safe
6. **Remove unnecessary setup/teardown** - Only test what's needed

## Validation

```bash
time npm test  # Should complete in < 30 seconds with all tests passing
```