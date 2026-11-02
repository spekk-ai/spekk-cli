---
id: property-sections-are-indexed
parent: property-format
created: 2026-11-02T21:30:00Z
priority: 1
status: done
branch: feat/v2-property-format
---

# The Terms, Decisions and Rationale of a Property Are in the Index

## Description

The terms, decisions and rationale hold the intent that is not in the code. An agent must find them with one query, and not by a read of the whole file.

## Success criteria

- `.spekk/index.db` has a table `property_sections(assertion_id, section, text)`.
- An assertion body with a level-2 heading `## Terms`, `## Decisions` or `## Rationale` has one row for each such section. The `section` value is `terms`, `decisions` or `rationale`. The heading match ignores letter case.
- The `text` is the body of the section: the lines after the heading, up to the next heading of level 1 or 2, or the end of the file. Leading and trailing blank lines are removed. A level-3 heading inside the section is part of the text.
- An assertion with none of these sections has no rows. This is the usual case for a v1 assertion.
- A parent spec file never adds rows. The sections belong to properties only.
- `spekk query "SELECT text FROM property_sections WHERE assertion_id = 'x' AND section = 'decisions'"` returns the decisions of `x`.

**Tests:** `internal/parser/property_format_test.go`, `internal/index/property_format_test.go`.
