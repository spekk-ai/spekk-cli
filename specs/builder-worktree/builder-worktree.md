---
id: builder-worktree
created: 2026-02-18T14:00:00Z
priority: 1
---

# Builder Worktree Isolation

## Overview

The builder agent works in isolated git worktrees to avoid conflicts with spec editing. Each parent spec gets its own worktree where the builder implements all assertions for that spec.

This allows the coach (or user) to create and edit specs in the main repo while the builder works in parallel.

## Why Worktrees

**Problem:**
- Builder is implementing an assertion (modifying files)
- User wants to create a new spec or edit existing specs
- Both working in same directory → conflicts, interruptions

**Solution:**
- Builder works in `feature/<spec-id>/` worktree
- Main repo stays clean for spec editing
- PRs contain both implementation AND spec status updates

## Workflow

```
1. spekk next → returns assertion from spec "user-dashboard"

2. Builder checks: does worktree exist for this spec?
   └── No: git worktree add feature/user-dashboard -b feature/user-dashboard

3. Builder works in feature/user-dashboard/
   └── Implements assertion
   └── Updates spec status to done

4. Builder continues with next assertion in same spec
   └── All assertions for "user-dashboard" done in same worktree

5. Builder creates PR from worktree branch

6. PR merges → main has implementation + done statuses

7. Builder cleans up worktree
   └── git worktree remove feature/user-dashboard
```

## Worktree Structure

```
repo/                           ← main, coach edits specs here
├── specs/
├── src/
└── feature/
    └── user-dashboard/         ← worktree for spec
        ├── specs/              ← builder updates status here
        └── src/                ← builder implements here
```

## Edge Cases

### Coach edits spec while builder is working on it

**Status:** Unresolved - need to determine approach

Options:
1. Builder pulls from main before creating PR (gets coach's changes)
2. Coach sees warning if spec has assertions "in_progress"
3. Conflicts resolved manually during PR review

### Multiple specs in progress

Builder can have multiple worktrees for different specs:
```
feature/user-dashboard/     ← working on this
feature/order-management/   ← also in progress
feature/auth-flow/          ← also in progress
```

Each is independent, each becomes its own PR.

### Worktree cleanup

After PR merges:
1. Builder detects merged branch
2. Removes worktree: `git worktree remove feature/<spec-id>`
3. Deletes local branch: `git branch -d feature/<spec-id>`

## Integration with CLI

The `spekk` CLI needs commands for worktree management:

```bash
# Builder uses internally
spekk worktree create <spec-id>   # Create worktree for spec
spekk worktree status             # List active worktrees
spekk worktree cleanup            # Remove merged worktrees

# Or automatic - builder handles this internally
```

## Benefits

1. **No conflicts**: Builder and coach work in separate directories
2. **Clean PRs**: Implementation + spec updates in one PR
3. **Parallel work**: Multiple specs can be in progress
4. **Easy rollback**: Delete worktree to abandon work
5. **Review-friendly**: Each spec is a reviewable unit
