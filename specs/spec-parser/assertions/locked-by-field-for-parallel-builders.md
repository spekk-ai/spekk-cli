---
id: locked-by-field-for-parallel-builders
parent: spec-parser
created: 2026-02-25T19:30:00Z
priority: 1
status: not_started
---

# Locked-By Field for Parallel Builders

## What Must Be True

Assertions support a `locked-by` field that enables parallel builders to claim work without conflicts.

## Success Criteria

- ✅ Parser reads optional `locked-by` field from assertion YAML frontmatter
- ✅ Parser exposes `lockedBy` in assertion JSON output
- ✅ `spekk next` skips assertions where `status: in_progress` AND `locked-by` is set
- ✅ `spekk next` includes assertions where `status: in_progress` but no `locked-by` (backwards compatible)
- ✅ Builder sets `locked-by` field when marking assertion `in_progress`
- ✅ Builder removes `locked-by` field when marking assertion `done` or `failed`
- ✅ Lock format: `locked-by: builder-{hostname}-{pid}-{timestamp}`
- ✅ Stale locks (>2 hours old) are ignored by `spekk next`

## Workflow

**Builder claims work:**
1. Run `spekk next` → get assertion without `locked-by`
2. Update YAML frontmatter:
   ```yaml
   status: in_progress
   locked-by: builder-macbook-12345-1706210400
   ```
3. Commit immediately
4. Pull to check for conflicts
5. If conflict (someone else claimed it), resolve and pick next assertion

**Builder releases lock:**
1. Work completes
2. Update YAML frontmatter:
   ```yaml
   status: done
   # locked-by field removed
   ```
3. Commit

**Parallel safety:**
- Builder A claims assertion-1
- Builder B runs `spekk next` → gets assertion-2 (assertion-1 is locked)
- No conflicts!

## Lock Format

```
locked-by: builder-{hostname}-{pid}-{timestamp}
```

Example: `builder-macbook-pro-98765-1706210400`

Components:
- `builder-` prefix (for clarity)
- Hostname (identifies machine)
- Process ID (uniqueness within machine)
- Unix timestamp (for stale lock detection)

## Stale Lock Detection

If current time - lock timestamp > 2 hours (7200 seconds):
- Lock is considered stale
- `spekk next` ignores the lock
- Allows recovery from crashed builders

## Implementation Notes

- Parser already handles arbitrary frontmatter fields
- Add `lockedBy` to parser output alongside `status`, `priority`, etc.
- Builder needs to:
  - Generate lock ID on startup
  - Write lock when claiming work
  - Remove lock on completion
  - Handle git conflicts gracefully
- Backwards compatible: assertions without `locked-by` work as before
