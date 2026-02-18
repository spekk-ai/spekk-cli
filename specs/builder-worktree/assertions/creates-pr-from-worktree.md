---
id: creates-pr-from-worktree
parent: builder-worktree
created: 2026-02-18T14:00:00Z
priority: 1
status: not_started
---

# Builder Creates PR from Worktree

## What Must Be True

When all assertions in a spec are done (or builder is ready to submit), it creates a PR from the worktree branch.

## PR Contents

The PR includes everything done in the worktree:

```
PR: feat(user-dashboard): implement user dashboard spec
├── src/components/Dashboard.tsx (new)
├── src/components/RecentOrders.tsx (new)
├── src/services/dashboardService.ts (new)
├── tests/Dashboard.test.tsx (new)
├── specs/user-dashboard/assertions/shows-welcome.md (status: done)
├── specs/user-dashboard/assertions/displays-orders.md (status: done)
└── specs/user-dashboard/assertions/has-quick-actions.md (status: done)
```

## When to Create PR

Options:
1. **All assertions done**: Wait until every assertion in spec is complete
2. **Manual trigger**: Builder asks user or user runs command
3. **Per-session**: Create PR at end of builder session

Recommended: Create PR when all assertions in spec are done, or when user explicitly requests.

## PR Creation Command

```bash
# From worktree directory
gh pr create \
  --title "feat(<spec-id>): implement <spec-title>" \
  --body "$(generate_pr_body)"
```

## PR Body Format

```markdown
## Summary

Implements the <spec-title> spec.

## Assertions Completed

- [x] shows-welcome-message
- [x] displays-recent-orders
- [x] has-quick-actions

## Test Plan

- [ ] Unit tests pass
- [ ] E2E tests pass (if applicable)
- [ ] Manual verification of success criteria

---
Spec: specs/<spec-id>/<spec-id>.md
```

## Success Criteria

- PR created from worktree branch
- PR title references spec
- PR body lists completed assertions
- PR includes all implementation + spec status updates
- PR targets main branch
