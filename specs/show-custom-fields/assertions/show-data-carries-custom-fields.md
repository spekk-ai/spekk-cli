---
id: show-data-carries-custom-fields
parent: show-custom-fields
created: 2026-11-19T14:55:40Z
priority: 2
status: done
branch: feature/show-custom-fields
---

# Specs and Assertions in the Show Data Carry Their Custom Frontmatter Fields

## Success Criteria

- Each spec and assertion object in the show data has a `customFields` key: an object mapping each custom frontmatter key to an array of string values.
- Values are split by the same rule as `frontmatter_fields`: one item per value, brackets and per-item quotes stripped, whitespace trimmed. Scalar, flow-sequence, and flat block-list spellings of the same values produce the same array. Block scalars (`key: |` / `key: >`) are excluded.
- A repeated value appears once, matching the one row per distinct value in `frontmatter_fields`.
- Known keys never appear in `customFields` (for example `id`, `status`, `priority`, `depends-on`, `branch`).
- `customFields` is omitted from an item that has no custom fields, so output for a tree without custom fields is byte-identical to output before this change.
- `spekk show --json` and the data embedded in the HTML of `spekk show` contain the same `customFields` for one fixture tree. A test compares them.
- In `--cross-branch` mode, an item that exists only on another branch (`foreign: true`) carries the custom fields read from that branch's file.

**Note:** The split must come from the parser's existing custom-field rule, not a second copy of it. A value that indexes as two rows shows as two array items.

**Tests:** `internal/show/custom_fields_test.go`
