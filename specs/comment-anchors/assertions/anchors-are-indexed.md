---
id: anchors-are-indexed
parent: comment-anchors
created: 2026-11-02T21:30:00Z
priority: 1
status: not_started
depends-on: comment-anchor-ids-exist
---

# The Index Holds Every Anchor With Its File and Line

## Description

The `anchors` table holds the comment anchors and the front-matter anchors of each property, so that `spekk constraints` and the truth calculation read one table.

## Success criteria

- After `spekk index`, the `anchors` table has one row `(assertion_id, path, line, source)` for each comment anchor, with `source = 'comment'`, the project-relative path, and the 1-based line number.
- The rows from front-matter anchors (`frontmatter-anchors-name-files`) are in the same table with `source = 'frontmatter'`.
- An anchor with an unknown ID adds no row.
- The index stores the HEAD commit of its build. A reader that calls `EnsureFresh` rebuilds the index when HEAD is now a different commit. Thus the anchors of a new commit show with no manual `spekk index`.
- `spekk query "SELECT path, line FROM anchors WHERE assertion_id = 'x'"` returns every anchor of `x`.
