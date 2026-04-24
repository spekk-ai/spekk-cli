---
id: builder-agent
created: 2026-01-20T18:15:00Z
priority: 1
---

# Builder Agent (SUPERSEDED)

> **Superseded:** This spec describes the original Node.js implementation. See `golang-agents` for the Go replacement.

## Overview

The builder agent is the execution engine of the spec-driven system. It takes assertions and makes them true through implementation and testing.

This is the core "work loop" that turns declarative specs into working code.

## What It Does

### 1. Get Next Task
Runs parser to identify highest-priority incomplete assertion:
```bash
node scripts/next-spec.js
```

### 2. Read Assertion
Understands what must be true:
- Success criteria
- Testability (can this be automated?)
- Context from parent spec

### 3. Determine Testability
- **Code/scripts** → YES (write unit tests)
- **UI/UX** → MAYBE (integration tests)
- **Manual process** → NO (prose validation)

### 4. Write Tests First (if testable)
- Create test file in `tests/`
- Link test in assertion: `**Tests:** tests/file.test.js`
- Write tests that validate success criteria

### 5. Implement
Make tests pass or satisfy success criteria

### 6. Validate
- Run tests (`npm test`)
- Verify no regressions
- Check success criteria met

### 7. Update Status
Edit assertion frontmatter:
```yaml
```

### 8. Commit
Create git commit with changes

## Integration with System

**Triggered by:** Ralph loop after parser identifies next work

**Reads from:**
- Parser JSON output
- Assertion markdown files
- Parent spec files (for context)

**Writes to:**
- Implementation files (code, scripts, etc.)
- Test files (if testable)
- Assertion status updates

**Output:** Working implementation with passing tests

## Why Builder Is Priority 1

The builder IS the system in bootstrap phase:
1. Without builder, nothing gets implemented
2. Builder implements parser (which drives everything)
3. Builder implements itself (self-hosting)
4. Other agents come later

The builder must work before we can be truly spec-driven.

## Prompt

See `builder-agent.prompt.md` for detailed agent instructions.

## Assertions

See `assertions/` for what must be true about builder behavior.
