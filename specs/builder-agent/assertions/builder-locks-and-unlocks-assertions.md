---
id: builder-locks-and-unlocks-assertions
parent: builder-agent
created: 2026-02-25T20:10:00Z
priority: 1
status: done
depends-on: locked-by-field-for-parallel-builders
branch: feature/builder-locks
---

# Builder Locks and Unlocks Assertions

## What Must Be True

The builder agent locks assertions when starting work and releases locks when completing work, enabling parallel builder execution.

## Success Criteria

- ✅ Builder adds `locked-by` field when marking assertion `in_progress`
- ✅ Lock format: `builder-{hostname}-{pid}-{timestamp}`
- ✅ Builder commits lock immediately after adding (before starting implementation)
- ✅ Builder pulls after committing to detect conflicts
- ✅ If conflict detected (someone else locked it), builder picks next assertion
- ✅ Builder removes `locked-by` field when marking assertion `done`
- ✅ Builder removes `locked-by` field when marking assertion `failed`
- ✅ Lock removal happens in same commit as status change
- ✅ Builder prompt includes locking instructions

## Lock Workflow

**Claiming work:**
1. Run `spekk next` → get unlocked assertion
2. Update YAML frontmatter:
   ```yaml
   status: in_progress
   locked-by: builder-macbook-pro-12345-1706210400
   ```
3. Commit immediately with message: "Lock assertion: {assertion-id}"
4. Pull to check for conflicts
5. If conflict:
   - Resolve (accept other builder's lock)
   - Run `spekk next` again to get different assertion
6. If no conflict:
   - Begin implementation

**Releasing lock:**
1. Work completes
2. Update YAML frontmatter:
   ```yaml
   status: done
   # locked-by field removed
   ```
3. Commit with implementation changes

## Generating Lock ID

Builder generates lock ID on startup:
```bash
HOSTNAME=$(hostname)
PID=$$
TIMESTAMP=$(date +%s)
LOCK_ID="builder-${HOSTNAME}-${PID}-${TIMESTAMP}"
```

Example: `builder-macbook-pro-12345-1706210400`

## Implementation Notes

- Builder prompt already includes locking instructions (section 5)
- Parser already handles `locked-by` field (see locked-by-field-for-parallel-builders)
- Git provides atomic commit mechanism for claiming work
- Stale locks (>2 hours) handled by parser, not builder
- Builder doesn't need to check lock age - parser does it

**Tests:** `src/builder/__tests__/builder-locks-and-unlocks.test.js`

## Validation

Test parallel builders:
1. Run builder A on assertion-1
2. Run builder B simultaneously
3. Verify:
   - Both builders lock different assertions
   - No duplicate work
   - Locks released on completion
