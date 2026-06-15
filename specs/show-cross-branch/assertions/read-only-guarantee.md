---
id: read-only-guarantee
parent: show-cross-branch
created: 2026-06-15T12:00:00Z
priority: 1
status: not_started
branch: feat/show-cross-branch
---

# Cross-Branch Mode Never Mutates the Working Tree or Index

## Description

The entire cross-branch flow is a read-only preview. It must never change the
working tree, the index, the current branch, or any refs. This is a hard
constraint that protects users running the explorer mid-work.

## Success Criteria

- The cross-branch flow uses only read-only / in-memory git operations:
  `git --version`, ref discovery (`for-each-ref` / `branch`), `git merge-base`,
  `git diff`/`diff-tree`, `git show <ref>:<path>`, and
  `git merge-tree --write-tree` (which writes to the object store only, not the
  working tree or index).
- The flow **never** runs `git checkout`, `git switch`, `git merge`,
  `git reset`, `git stash`, `git add`, or any other command that modifies the
  working tree, index, current branch, or refs.
- Running `spekk show --cross-branch` leaves `git status` output, the current
  branch (`HEAD`), and the working tree exactly as they were before the command.
- This guarantee holds even when the working tree has uncommitted changes.
