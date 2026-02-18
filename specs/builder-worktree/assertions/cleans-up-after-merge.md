---
id: cleans-up-after-merge
parent: builder-worktree
created: 2026-02-18T14:00:00Z
priority: 2
status: not_started
---

# Builder Cleans Up Worktree After PR Merges

## What Must Be True

After a PR is merged, the builder cleans up the worktree and local branch to avoid clutter.

## Cleanup Steps

```bash
# 1. Check if PR is merged
gh pr view feature/<spec-id> --json state -q '.state'
# Returns: "MERGED"

# 2. Remove worktree
git worktree remove feature/<spec-id>

# 3. Delete local branch
git branch -d feature/<spec-id>

# 4. Optionally prune remote tracking
git fetch --prune
```

## When to Clean Up

Options:
1. **Automatic**: Builder checks for merged PRs at start of session
2. **Manual**: User runs `spekk worktree cleanup`
3. **On next run**: When builder starts, clean up any merged worktrees

Recommended: Automatic check at start of builder session.

## Listing Active Worktrees

```bash
git worktree list
```

Output:
```
/path/to/repo                   abc1234 [main]
/path/to/repo/feature/user-dashboard  def5678 [feature/user-dashboard]
/path/to/repo/feature/auth-flow       ghi9012 [feature/auth-flow]
```

## Success Criteria

- Merged worktrees are detected
- Worktree directory removed after merge
- Local branch deleted after merge
- No orphaned worktrees accumulate
- Cleanup doesn't affect unmerged worktrees
