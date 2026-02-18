---
id: auto-creates-worktree-per-spec
parent: builder-worktree
created: 2026-02-18T14:00:00Z
priority: 1
status: not_started
---

# Builder Auto-Creates Worktree Per Spec

## What Must Be True

When builder gets an assertion to work on, it must automatically create a worktree for that spec if one doesn't exist.

## Behavior

```
1. Builder runs `spekk next`
   → Returns assertion with parent spec ID

2. Builder checks for existing worktree
   → Look for feature/<spec-id>/

3. If no worktree exists:
   → git worktree add feature/<spec-id> -b feature/<spec-id>
   → cd into worktree
   → npm install (if needed)

4. If worktree exists:
   → cd into existing worktree
   → git pull origin main (get latest spec changes)

5. Builder works in worktree
```

## Worktree Location

Worktrees are created as siblings to the main repo:

```
/path/to/repo/                    ← main repo
/path/to/repo/feature/<spec-id>/  ← worktree
```

Or under a `feature/` directory:
```
/path/to/repo/
└── feature/
    └── <spec-id>/                ← worktree
```

## Commands Used

```bash
# Create worktree with new branch
git worktree add feature/<spec-id> -b feature/<spec-id>

# Or if branch exists
git worktree add feature/<spec-id> feature/<spec-id>

# List worktrees
git worktree list
```

## Success Criteria

- Builder creates worktree automatically when starting work on a spec
- Worktree named `feature/<spec-id>`
- Branch named `feature/<spec-id>`
- Builder continues work in worktree, not main repo
- If worktree exists, builder reuses it (doesn't recreate)
- Dependencies installed in worktree if package.json exists
