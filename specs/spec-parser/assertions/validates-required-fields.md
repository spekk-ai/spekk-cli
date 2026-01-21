---
id: validates-required-fields
parent: spec-parser
created: 2026-01-20T16:10:00Z
priority: 2
status: not_started
---

# Parser Must Validate Required Fields

## What Must Be True

The parser must validate that all spec and assertion files have required frontmatter fields:

### Required for Specs
- `id` (string, kebab-case, unique)
- `created` (string, ISO 8601 timestamp)
- `priority` (integer, 0 or greater)

### Required for Assertions
- `id` (string, kebab-case, unique within parent)
- `parent` (string, must match existing spec id)
- `created` (string, ISO 8601 timestamp)
- `priority` (integer, 0 or greater)

### Optional Fields
- `status` (string: not_started | in_progress | done) - defaults to `not_started`
- `updated` (string, ISO 8601 timestamp)

## Validation Rules

**For `id` field:**
- Must be kebab-case (lowercase with hyphens)
- No spaces, underscores, or special characters
- Must be unique across all specs (for spec files)
- Must be unique within parent spec (for assertion files)
- Examples: `spec-parser`, `validates-required-fields`

**For `created` and `updated` fields:**
- Must be valid ISO 8601 format
- Must be UTC timezone (ending in Z)
- Format: `YYYY-MM-DDTHH:MM:SSZ`
- Example: `2026-01-20T16:10:00Z`

**For `priority` field:**
- Must be integer: 1, 2, or 3
- 1 = highest priority (critical, blocking)
- 2 = medium priority (important, not blocking)
- 3 = lowest priority (nice to have, future)
- No other values allowed

**For `status` field:**
- If present, must be one of: `not_started`, `in_progress`, `done`
- If missing, defaults to `not_started`

**For `parent` field (assertions only):**
- Must reference an existing spec `id`
- Assertion must be located in `specs/{parent}/assertions/`

## Error Handling

Parser should report clear errors:
- "Missing required field 'id' in specs/foo/bar.md"
- "Invalid ISO 8601 timestamp in 'created' field: '2026-01-20'"
- "Duplicate spec id 'spec-parser' found in multiple files"
- "Invalid status value 'completed' (must be: not_started, in_progress, done)"
- "Invalid priority value '0' (must be: 1, 2, or 3)"
- "Parent spec 'unknown-spec' not found for assertion 'validates-required-fields'"

## Success Criteria

- ✅ Parser rejects files missing required fields
- ✅ Parser validates field formats
- ✅ Parser detects duplicate IDs
- ✅ Parser validates parent references for assertions
- ✅ Parser provides clear, actionable error messages

**Tests:** `tests/spec-parser.test.js`
