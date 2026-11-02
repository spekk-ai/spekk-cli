---
id: frontmatter-anchors-name-files
parent: property-format
created: 2026-11-02T21:30:00Z
priority: 1
status: not_started
---

# Each Front-Matter Anchor Names a File That Exists

## Description

A property with no checks can list the files that hold its code in an `anchors` front-matter key. A stale anchor is worse than no anchor: an agent trusts it and does not read the code. Thus `spekk validate` rejects an anchor that names no file.

## Success criteria

- `anchors` accepts the list spellings that `frontmatter_fields` accepts: a flow list (`[a.py, b.py]`), a comma-separated scalar, and a block list.
- Each anchor is a path relative to the project root (the directory that holds `specs/`).
- `spekk validate` fails, with one line that names the assertion file and the anchor, for each anchor that is an absolute path, contains a `..` segment, names a directory, or names no file in the working tree.
- After `spekk index`, the `anchors` table has one row for each front-matter anchor: `(assertion_id, path, line, source)` with `source = 'frontmatter'` and `line` NULL. `comment-anchors` adds the rows with `source = 'comment'` to the same table.
- An assertion with no `anchors` key has no front-matter rows, and validates as in v1.
