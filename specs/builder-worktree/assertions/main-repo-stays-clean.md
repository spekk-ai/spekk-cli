---
id: main-repo-stays-clean
parent: builder-worktree
created: 2026-02-18T14:00:00Z
priority: 1
status: not_started
---

# Main Repo Stays Clean for Spec Editing

## What Must Be True

The main repo (main branch) remains untouched by builder implementation work. This allows coach/user to freely create and edit specs without conflicts.

## What Main Repo Is For

- Creating new specs (coach)
- Editing existing specs (coach/user)
- Viewing spec status
- Running `spekk next` to see what's available
- General repo administration

## What Main Repo Is NOT For

- Implementation code changes (done in worktree)
- Spec status updates (done in worktree, merged via PR)
- Test file changes (done in worktree)

## Behavior

```
User in main repo:
  $ spekk coach
  → Creates new spec: specs/new-feature/new-feature.md
  → No conflicts with builder

Builder (in background):
  → Working in feature/user-dashboard/ worktree
  → Modifying src/, tests/, specs status
  → Completely isolated from main

User can:
  $ git status
  → Clean (only their spec changes)
  → No builder work-in-progress files
```

## Success Criteria

- Builder never modifies files in main repo directly
- Builder always works in worktree
- Main repo `git status` shows only user's changes
- Coach can create/edit specs while builder runs
- No merge conflicts from simultaneous work
