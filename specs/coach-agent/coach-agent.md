---
id: coach-agent
created: 2026-01-20T17:00:00Z
priority: 3
---

# Coach Agent (SUPERSEDED)

> **Superseded:** This spec describes the original Node.js implementation. See `golang-agents` for the Go replacement.

## Overview

The coach agent translates messy, imperative user requests into clean, testable declarative specs. This is the "front door" of the spec-driven system - where requirements emerge through conversation and refinement.

## Critical Constraint: Code is Read-Only

**The coach NEVER writes or edits implementation code.**

- ⛔ Coach does NOT use Edit or Write tools on `.js`, `.ts`, `.jsx`, `.tsx` files
- ⛔ Coach does NOT modify files in `app/` directory
- ✅ Coach CAN read code to understand context for writing better specs
- ✅ Coach ONLY writes spec files (`.md` files in `specs/` directory)

**Role separation:**
- **Coach:** Writes specs describing WHAT must be true
- **Builder:** Writes code to MAKE it true

If a user asks the coach to fix code directly, coach responds: "I'm the coach - I write specs. The builder agent implements them."

## The Problem It Solves

Users don't naturally think in terms of testable assertions. They say:
- "We should fix the copy on the login page"
- "Add dark mode"
- "Make the dashboard faster"

The coach helps refine these into well-formed specs with clear, testable assertions.

## What It Does

### 1. Receive User Request
Accept natural language input about what the user wants to build or change.

### 2. Check Existing Specs
- **Receive full spec JSON hierarchy** (output from parser `--all` flag)
- Scan existing specs to see if this is an update or new spec
- If updating existing spec, load it for context
- Identify potential conflicts with existing specs

**Context requirement:**
Coach must receive the complete spec hierarchy on every invocation:
```json
{
  "type": "hierarchy",
  "specs": [
    {
      "id": "spec-parser",
      "title": "Spec Parser",
      "priority": 1,
      "status": "in_progress",
      "assertions": [...]
    },
    ...
  ]
}
```

This provides full context of:
- All existing specs and their assertions
- Current status of each item
- Priority levels
- What's been built, what's in progress, what's not started

Without this context, coach can't make informed decisions about where new specs fit or if they conflict with existing work.

### 3. Ask Clarifying Questions
Guide the user through refinement:
- What exactly should happen?
- How will we know it's working?
- What's the priority? (1=critical, 2=important, 3=nice-to-have)
- Are there edge cases to consider?
- Should this be broken into multiple assertions?

### 4. Draft Spec Structure
Propose a spec with:
- Clear spec ID (kebab-case)
- Main spec file describing the feature
- Specific, testable assertions
- Appropriate priority levels
- Success criteria for each assertion

### 5. Get User Approval
Show the proposed spec structure and ask if it captures their intent.

### 6. Create or Update Spec Files
Write the spec and assertion files to `specs/` directory with proper:
- Folder structure (`specs/{spec-id}/`)
- YAML frontmatter (id, created, priority, status)
- Clear markdown content
- Testability indicators (can this be automated?)

**Preserve design artifacts:**
When users provide mockups, prototypes, or design references:
- Save mockup images as `specs/{spec-id}/mockup.png` (or appropriate extension)
- Save prototype HTML/CSS files alongside the spec
- Reference artifacts using relative paths in the spec
- This keeps requirements and their references together for future reference

**Status management when updating specs:**
- If creating new spec/assertion: `status: not_started`
- If updating existing assertion with `status: done`:
  - Change to `status: in_progress`
  - This signals builder to re-implement
  - Ensures updated requirements get picked up
- If updating assertion already `in_progress` or `not_started`: keep as-is

### 7. Commit Spec Changes
Immediately commit spec changes to git with clear, descriptive messages. This creates an audit trail of spec evolution and helps track when requirements changed.

Commit message should describe what the spec defines or what changed.

### 8. Re-coordinate (for feature branch changes)
After updating assertions on a feature branch, re-run the coordinator skill to refresh the dependency tree and work plan.

**When to re-coordinate:**
- Adding new assertions to a feature branch
- Changing assertion priorities or dependencies
- Marking assertions as failed (requires re-work)
- Major structural changes to branch

**Process:**
1. Read `specs/coach-skills-system/coordinator-skill.md`
2. Analyze current branch dependency structure
3. Show updated dependency tree to user
4. Validate with parser
5. Update frontmatter if dependencies changed
6. Commit coordination updates if needed

**Why this matters:**
- Keeps dependency tree accurate for builders
- Identifies newly-unlocked parallel work
- Prevents conflicts from stale dependencies
- Shows clear path through feature work

**Skip if:**
- Only content changes (not status/priority/dependencies)
- Working on isolated main branch assertions
- No structural changes to dependencies

## Example Interaction

```
User: "We need to add dark mode to the dashboard"

Coach: I can help spec that out. Let me ask a few questions:

1. Should dark mode be:
   - User preference (toggle in settings)
   - System preference (auto-detect OS setting)
   - Both?

2. Which components need dark mode support?
   - Just the dashboard?
   - Entire application?

3. What's the priority?
   - 1 (critical - blocks other work)
   - 2 (important - should do soon)
   - 3 (nice to have - when there's time)

User: User preference, entire application, priority 2

Coach: Perfect. Here's what I'm proposing:

Spec: dark-mode-support
Priority: 2

Assertions:
1. User can toggle dark mode in settings (priority 1)
2. All components render correctly in dark mode (priority 1)
3. Dark mode preference persists across sessions (priority 2)
4. Smooth transition animation between modes (priority 3)

Does this capture what you want?

User: Yes

Coach: ✅ Created spec at specs/dark-mode-support/
Next: Builder agents will implement these assertions in priority order.
```

## Output Format

The coach creates files in the specs directory:

```
specs/
└── dark-mode-support/
    ├── dark-mode-support.md       # Main spec
    └── assertions/
        ├── user-can-toggle.md
        ├── components-render.md
        ├── preference-persists.md
        └── smooth-transition.md
```

## Integration with Builder Workflow

**Creating new specs:**
1. Coach creates spec with assertions (`status: not_started`)
2. Parser identifies next assertion to work on
3. Builder agent implements it
4. Tests validate it
5. Builder marks `status: done`
6. Repeat

**Updating existing specs:**
1. Coach updates assertion content
2. Coach changes `status: done` → `status: in_progress`
3. Parser identifies assertion needs work
4. Builder re-implements with new requirements
5. Tests validate updated behavior
6. Builder marks `status: done` again

This status workflow ensures changes to specs trigger re-implementation.

## Why This Matters

**Without coach:**
- Users write specs incorrectly
- Specs are too vague or too specific
- Assertions aren't testable
- Priority is unclear

**With coach:**
- Consistent spec format
- Clear, testable assertions
- Appropriate granularity
- Proper prioritization
- Users guided through best practices

This is the **imperative → declarative** bridge that makes the system usable.

## Declarative vs Imperative

**Critical distinction:** Assertions must be DECLARATIVE, not imperative.

**Imperative (wrong):**
- "Migrate code to app/"
- "Move parser to app/parser/"
- "Refactor the dashboard"
- Instructions about WHAT TO DO

**Declarative (correct):**
- "No implementation code exists outside app/"
- "Parser implementation lives in app/parser/"
- "Dashboard displays spec hierarchy"
- Statements about WHAT MUST BE TRUE

**Why this matters:**
- User says: "We should move the parser code" (imperative)
- Coach translates to: "Parser implementation lives in app/parser/" (declarative)
- Builder reads assertion and figures out HOW to make it true
- Assertion describes target state, not the steps

**Coach's job:** Help users express their imperative requests as declarative assertions about desired state.

## Repository Hygiene Principle

The coach ensures specs maintain good repository hygiene. Generated files and dependencies should never be committed.

**What should be git-ignored:**
- Dependencies (`node_modules/`) - reproducible from package manifests
- Build artifacts (`dist/`, `build/`, `out/`) - reproducible from source
- Secrets (`.env`) - never commit credentials
- Generated files - derived from source, not source of truth

**Coaching approach:**
- `.gitignore` is itself a form of specification - respect and trust it
- When creating specs, express intent ("build artifacts are not committed") without duplicating .gitignore patterns
- Guide users to think about source vs. derived files
- Specs should declare what files/artifacts exist, not specify .gitignore rules

This principle keeps the repository clean and focused on source of truth.

## Assertions

See `assertions/` subfolder for what must be true for the coach agent to work correctly.
