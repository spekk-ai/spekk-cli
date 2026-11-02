---
id: constraints-lists-anchored-properties
parent: constraints-command
created: 2026-11-02T21:30:00Z
priority: 2
status: not_started
depends-on: anchors-are-indexed
---

# `spekk constraints <file>` Lists Every Property That Anchors on the File, and No Other

## Description

The list is complete, because a property that is not in it is a rule that the agent does not know it can break.

## Success criteria

- The output has one row for each property with a comment anchor in the file or a front-matter anchor that names the file. A property with anchors of both kinds in the file has one row.
- Each row has the property ID, the workflow value, the truth at HEAD with the level (as in `list-shows-truth-and-level`), the line numbers of its comment anchors in the file (empty for a front-matter anchor), and the title.
- Rows sort by priority, then by ID, as `spekk list` does.
- `--json` prints an array of objects with the keys `id`, `status`, `truth`, `level`, `lines` and `title`. With no match it prints `[]`.
- The file can be given relative to the current directory or as an absolute path. Spekk matches it as a path relative to the project root. A path outside the project root exits 1 with an error.
- A file with no anchors prints nothing on stdout and `no properties anchor on <path>` on stderr, and exits 0.
- `spekk constraints` is listed in `spekk help`, and `spekk constraints --help` exits 0.
