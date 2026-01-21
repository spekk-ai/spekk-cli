---
id: builder-runner-exists
parent: builder-agent
created: 2026-01-20T19:40:00Z
priority: 1
status: done
---

# Builder Runner Script Exists

## What Must Be True

A builder runner script must exist at the project root that orchestrates the builder agent workflow: get next task → work on it → commit → repeat.

## File Location

```
builder-loop.sh      # Project root
```

## Expected Behavior

The script automates the builder agent workflow:

1. **Get next task** - Run `npm run next` to identify next assertion
2. **Work on task** - Invoke builder agent to implement the assertion
3. **Commit changes** - Create git commit when task completes
4. **Repeat** - Loop back to step 1

**Stop conditions:**
- No more incomplete assertions
- User interrupts (Ctrl+C)
- Builder encounters an error

## Script Responsibilities

**Task identification:**
- Call parser to get next highest-priority assertion
- Exit cleanly if no tasks remain

**Builder invocation:**
- Launch builder agent with appropriate context
- Pass assertion details to builder
- Capture builder output

**Commit automation:**
- Create meaningful commit messages
- Include assertion ID and title in commit
- Handle git operations safely

**Error handling:**
- Graceful failure if parser fails
- Stop loop if builder fails
- Clear error messages for user

## Usage

```bash
# Run the builder loop
./builder-loop.sh

# The script handles:
# - Finding next task
# - Running builder agent
# - Committing changes
# - Looping to next task
```

## Success Criteria

- ✅ `builder-loop.sh` exists at project root
- ✅ Script is executable (`chmod +x builder-loop.sh`)
- ✅ Script calls `npm run next` to get tasks
- ✅ Script invokes builder agent with task context
- ✅ Script creates git commits for completed work
- ✅ Script loops until no tasks remain
- ✅ Script handles Ctrl+C gracefully
- ✅ Script stops on errors (doesn't infinite loop)
- ✅ Clear output showing progress

**Tests:** Manual validation (integration script)

## Notes

This script is the automation layer that runs the builder agent repeatedly. It's expected to evolve as the builder workflow matures, hence it lives in the project root (not in app/) where it can be easily modified.
