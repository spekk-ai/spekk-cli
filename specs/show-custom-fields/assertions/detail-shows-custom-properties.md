---
id: detail-shows-custom-properties
parent: show-custom-fields
created: 2026-11-19T14:55:40Z
priority: 2
status: done
depends-on: show-data-carries-custom-fields
branch: feature/show-custom-fields
---

# The Detail View Lists Custom Fields Under "Properties"

## Success Criteria

- The detail view of a spec or of an assertion with custom fields shows a "Properties" block below the existing meta row (status, priority, file).
- Each custom field is one line: the key, then its values joined with `, `.
- Keys are sorted alphabetically (byte order).
- Keys and values are HTML-escaped. A value of `<script>alert(1)</script>` renders as text, and no script runs.
- An item with no custom fields renders no "Properties" block and no empty heading.
- The block stays legible at phone width: a long value wraps, and the page does not scroll horizontally.

**Tests:** `internal/show/properties_test.go`
