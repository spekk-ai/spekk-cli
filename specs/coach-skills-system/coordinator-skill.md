---
id: coordinator-skill
created: 2026-02-25T15:36:00Z
---

# Coordinator Skill

Analyzes draft/not_started assertions and creates a dependency-aware work plan with branch assignments.

**CLI Command:** `spekk coach coordinate`

## Triggers

- "plan the work"
- "organize branches"
- "dependency graph"
- "what can we build in parallel"
- "coordinate development"
- "scope the work"
- "branch strategy"
- "work plan"

## Workflow

1. **Read assertions**
   - Use `parseAllSpecs()` to read all assertions
   - Filter to `status: draft` or `status: not_started`

2. **Analyze dependencies**
   - For each assertion, determine what prerequisite functionality must exist first
   - Identify single-parent dependencies (one assertion depends on one other)
   - Group related assertions into feature clusters

3. **Assign branches**
   - Each cluster of related assertions → one feature branch
   - Generate semantic branch names (e.g., `feature/chat-system`, `feature/authentication`)
   - Isolated assertions → `main` branch

4. **Show plan to user**
   - Display dependency tree grouped by branch
   - Show which assertions can be built in parallel (different branches)
   - Show which assertions must be built serially (dependency chains)
   - Get user confirmation

5. **Update YAML frontmatter**
   - Add `depends-on: assertion-id` field (single parent, or omit if no dependency)
   - Add `branch: feature/name` field
   - Update all affected assertion files

6. **Validate with parser**
   - Run `spekk validate` to validate the updated structure
   - Parser will catch:
     - Invalid dependency IDs (non-existent assertions)
     - Circular dependencies
     - Invalid branch names
     - Malformed YAML
   - If parser reports errors, show them to user and abort
   - **Parse don't validate** - let the parser do its job

7. **Commit changes**
   - Single commit with all YAML updates
   - Commit message includes branch assignments and dependency chains
   - Git commit and show user next steps

## Validation

### Success Criteria
- User can see dependency tree before any changes
- User can approve or reject the plan
- YAML frontmatter is updated correctly
- Parser validates all changes (no manual validation needed)
- Single clean commit is created
- User receives clear next steps (create branches, start building)

### Example Output

```
Dependency Analysis
===================

feature/chat-system (4 assertions):
  websocket-connection (no dependencies)
    ↓
  chat-session-model (depends-on: websocket-connection)
    ↓
  chat-message-input (depends-on: chat-session-model)
  user-presence-tracking (no dependencies)

feature/audit-log (2 assertions):
  trial-search-api (no dependencies)
    ↓
  trial-eligibility-check (depends-on: trial-search-api)

main (2 assertions):
  update-button-styles (isolated, no dependencies)
  fix-header-typo (isolated, no dependencies)

Proceed with these updates? [y/N]
```

## Notes

- **Single-parent dependencies only** - Each assertion can depend on at most one other
- **Branch isolation** - Assertions on different branches should be independent
- **Sparse YAML** - Only add `depends-on` field when a dependency exists
- **Parser validation** - Use `parseAllSpecs()` to validate, don't reimplement validation
- **No API calls** - You (the coach) are the AI, use your intelligence directly
