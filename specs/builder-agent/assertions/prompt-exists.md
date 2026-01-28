---
id: prompt-exists
parent: builder-agent
created: 2026-01-20T19:30:00Z
priority: 1
status: done
---

# Builder Agent Prompt Must Exist

## What Must Be True

A clear, complete prompt file must exist at `specs/builder-agent/builder-agent.prompt.md` that defines how the builder agent operates.

## File Location

```
specs/
└── builder-agent/
    └── builder-agent.prompt.md
```

## Required Content

The prompt must include:

1. **Role definition** - What the builder agent does
2. **Workflow steps** - How to get tasks, implement, test, commit
3. **Stop instruction** - Work on ONE assertion at a time, then STOP
4. **Test requirements** - When and how to write tests
5. **Status management** - How to update assertion status
6. **File paths** - Updated references to `app/` directory structure (not `scripts/` or `tests/`)

## Key Requirements

**Directory references must be current:**
- ✅ References `app/parser/` for implementation
- ✅ References `app/**/__tests__/` for tests
- ✅ Uses `spekk next` command (global CLI, not `npm run next`)
- ❌ No references to `scripts/` directory
- ❌ No references to `tests/` directory

**Behavioral requirements:**
- Must explicitly state: "Work on ONE assertion at a time, then STOP"
- Must include step 7: "Stop" instruction
- Must not encourage looping or automatic continuation

## Success Criteria

- ✅ File exists at `specs/builder-agent/builder-agent.prompt.md`
- ✅ Contains role definition for builder agent
- ✅ Includes complete workflow (get task, read, implement, test, validate, update status, commit, stop)
- ✅ Explicitly instructs builder to work on ONE task and STOP
- ✅ References correct directory structure (`app/` not `scripts/` or `tests/`)
- ✅ Uses global CLI commands (`spekk next`, not local npm scripts)
- ✅ Explains status values (not_started, in_progress, done)
- ✅ Describes priority levels (1, 2, 3)

**Tests:** `specs/builder-agent/assertions/prompt-exists.test.sh`

Sidecar test should validate:
- File exists at correct path
- Contains key sections (role, workflow, stop instruction)
- References `app/` directory structure
- Uses `spekk next` command (global CLI)
- Does not reference deprecated `scripts/` or `tests/` directories
- Does not use local npm scripts for spec parsing
