---
id: mock-external-processes
parent: optimize-test-performance
created: 2026-01-28T21:25:00Z
priority: 1
status: not_started
---

# Mock External Processes

## What Must Be True

Tests use mocks for external processes instead of spawning real child processes, which are slow and unreliable.

## Current Problem

Tests like "Orchestration Loops" are spawning real processes:
- `spawn('node', ['--test', '--test-concurrency=1', ...testFiles])`
- Command executions that take 5+ seconds
- Real file system operations

## Success Criteria

- ✅ No `spawn()` or `exec()` calls in tests
- ✅ Mock child process responses
- ✅ Mock file system operations where appropriate
- ✅ Use in-memory fixtures instead of real files
- ✅ Tests run in isolation without external dependencies

## Implementation Strategy

**Mock child processes:**
```javascript
// Instead of real spawn
const mockSpawn = () => ({ on: jest.fn(), stdout: { on: jest.fn() } })
```

**Mock file system:**
```javascript
// Use in-memory file system or simple mocks
const mockFs = { readFile: jest.fn(), writeFile: jest.fn() }
```

**Use fixtures:**
```javascript
// Hardcode expected outputs instead of calling real commands
const expectedOutput = { specs: [...], assertions: [...] }
```