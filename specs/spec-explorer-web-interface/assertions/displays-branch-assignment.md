---
id: displays-branch-assignment
parent: spec-explorer-web-interface
created: 2026-02-25T18:01:00Z
priority: 1
status: draft
---

# Displays Branch Assignment

## What Must Be True

When an assertion has a `branch` field in its YAML frontmatter (and it's not `main`), the spec explorer displays the branch name in the tree view.

## Success Criteria

- ✅ Assertions with `branch` field (non-main) show branch badge
- ✅ Format: Small pill-shaped badge with branch icon and name
- ✅ Badge appears inline with status/priority badges
- ✅ Assertions on `main` or with no branch field show no badge
- ✅ Branch badge is visually distinct (purple/blue accent color)
- ✅ Badge text: "🌿 feature/name" or similar

## Example Visual

```
✅ chat-session-model (priority 1) 🌿 feature/chat-system
   → depends on: websocket-connection
```

## Implementation Notes

- Parser already exposes `branch` field from assertion YAML
- Add branch badge generation to HTML in `src/show/cli.js`
- Style similar to status/priority badges but with branch color scheme
- Use git branch emoji or simple text prefix
