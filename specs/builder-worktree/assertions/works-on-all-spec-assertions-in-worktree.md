---
id: works-on-all-spec-assertions-in-worktree
parent: builder-worktree
created: 2026-02-18T14:00:00Z
priority: 1
status: not_started
---

# Builder Works on All Spec Assertions in Same Worktree

## What Must Be True

Once a worktree is created for a spec, the builder works on ALL assertions for that spec in the same worktree. It does not create a new worktree per assertion.

## Why

- Too many worktrees is unmanageable
- Assertions in a spec are related (same feature area)
- Single PR for entire spec is cleaner to review
- Reduces git overhead

## Behavior

```
Spec: user-dashboard
├── assertion-1: shows-welcome-message (priority 1)
├── assertion-2: displays-recent-orders (priority 1)
└── assertion-3: has-quick-actions (priority 2)

Builder workflow:
1. Get assertion-1 → create feature/user-dashboard/ worktree
2. Implement assertion-1 in worktree
3. Mark assertion-1 done
4. Get assertion-2 → same worktree exists, reuse it
5. Implement assertion-2 in worktree
6. Mark assertion-2 done
7. Get assertion-3 → same worktree, reuse
8. Implement assertion-3 in worktree
9. Mark assertion-3 done
10. All assertions done → create PR
```

## Determining Spec from Assertion

The assertion frontmatter has `parent: <spec-id>`:

```yaml
---
id: shows-welcome-message
parent: user-dashboard        ← use this for worktree name
...
---
```

## Success Criteria

- Builder uses `parent` field to determine worktree name
- All assertions with same parent use same worktree
- Worktree contains cumulative changes for all assertions
- Single PR contains all assertion implementations
- Spec status updates for all assertions in same PR
