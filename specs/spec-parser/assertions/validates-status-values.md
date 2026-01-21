---
id: validates-status-values
parent: spec-parser
created: 2026-01-20T16:27:00Z
priority: 2
status: done
---

# Parser Must Validate Status Values

## What Must Be True

The parser must validate that `status` field (when present) contains only valid values.

## Valid Status Values

Only these four values are allowed:
- `not_started` (default if omitted)
- `in_progress`
- `done`
- `draft` (placeholder/planning, excluded from work queue)

## Validation Rules

### Default Behavior
- If `status` field is missing, treat as `not_started`
- Parser should not require explicit `status` field

### When Status is Present
- Must be exactly one of the four valid values
- Case-sensitive (must be lowercase with underscores)
- No other values allowed

## Invalid Examples

Parser should reject:
- `completed` (use `done` instead)
- `Not Started` (wrong case/format)
- `in-progress` (wrong separator, should be underscore)
- `todo` (not a valid status)
- `pending` (not a valid status)
- `finished` (use `done` instead)

## Error Messages

Parser should provide clear errors:
- "Invalid status value 'completed' (must be: not_started, in_progress, done, draft)"
- "Invalid status format 'Not Started' (must be lowercase with underscores)"

## Success Criteria

- ✅ Parser accepts all four valid status values (not_started, in_progress, done, draft)
- ✅ Parser treats missing status as `not_started`
- ✅ Parser rejects invalid status values
- ✅ Parser provides clear error messages for invalid status values
- ✅ Parser is case-sensitive for status validation
- ✅ Parser excludes 'draft' status from work queue (like 'done')