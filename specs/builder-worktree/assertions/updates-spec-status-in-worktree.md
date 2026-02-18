---
id: updates-spec-status-in-worktree
parent: builder-worktree
created: 2026-02-18T14:00:00Z
priority: 1
status: not_started
---

# Builder Updates Spec Status in Worktree

## What Must Be True

When builder completes an assertion, it updates the spec status in the worktree (not in main repo). The status change becomes part of the PR.

## Why

- PR is a complete unit: implementation + status update
- Main repo specs stay at `not_started` until PR merges
- Reviewers can see what assertions are being completed
- If PR is abandoned, main repo status is unaffected

## Behavior

```
Main repo:
  specs/user-dashboard/assertions/shows-welcome.md
    status: not_started  ← unchanged until merge

Worktree (feature/user-dashboard):
  specs/user-dashboard/assertions/shows-welcome.md
    status: done  ← updated by builder

PR diff shows:
  - status: not_started
  + status: done
```

## Success Criteria

- Builder edits spec files in worktree, not main repo
- Status changes from `not_started` → `in_progress` → `done`
- PR diff includes spec status changes
- Main repo specs unchanged until PR merges
- After merge, main repo has `status: done`
