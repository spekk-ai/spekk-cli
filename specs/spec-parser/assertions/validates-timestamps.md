---
id: validates-timestamps
parent: spec-parser
created: 2026-01-20T16:26:00Z
priority: 2
status: done
depends-on: parses-frontmatter
branch: feature/spec-parser
---

# Parser Must Validate Timestamp Formats

**Tests:** src/parser/__tests__/timestamp-validation.test.js

## What Must Be True

The parser must validate that `created` and `updated` fields use valid ISO 8601 format.

## Required Format

- **Format**: `YYYY-MM-DDTHH:MM:SSZ`
- **Timezone**: Must be UTC (ending in Z)
- **Example**: `2026-01-20T16:26:00Z`

## Validation Rules

### For `created` field:
- Required in all specs and assertions
- Must be valid ISO 8601 format
- Must be UTC timezone
- Immutable - never changes after creation

### For `updated` field:
- Optional field
- If present, must be valid ISO 8601 format
- Must be UTC timezone
- Should only be used for significant changes

## Invalid Examples

The parser should reject:
- `2026-01-20` (missing time)
- `2026-01-20T16:26:00` (missing timezone)
- `2026-01-20T16:26:00+00:00` (wrong timezone format)
- `Jan 20, 2026` (wrong format entirely)
- `invalid-date` (not a date)

## Error Messages

Parser should provide clear errors:
- "Invalid ISO 8601 timestamp in 'created' field: '2026-01-20'"
- "Missing timezone in timestamp: '2026-01-20T16:26:00'"
- "Invalid timestamp format in 'updated' field"

## Success Criteria

- ✅ Parser accepts valid ISO 8601 UTC timestamps
- ✅ Parser rejects invalid timestamp formats
- ✅ Parser provides clear error messages for invalid timestamps
- ✅ Parser validates both `created` and `updated` fields