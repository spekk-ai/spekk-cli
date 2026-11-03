---
id: constraints-reads-current-file
parent: constraints-command
created: 2026-11-02T21:30:00Z
priority: 2
status: done
branch: feat/v2-constraints-command
depends-on: constraints-lists-anchored-properties
---

# `spekk constraints` Reads the Comment Anchors From the File as It Is Now

## Description

The index stores the anchors of its last build. An agent can add or move an anchor and then ask again before it commits. The answer must not come from a stale index.

## Success criteria

- The comment anchors come from a scan of the file on disk at the time of the command, not from the `anchors` table.
- After an edit that adds `// spekk: <id>` to a file, `spekk constraints <file>` lists `<id>` with no `spekk index` and no commit.
- After an edit that removes the last anchor of a property from the file, the property is no longer in the list, unless a front-matter anchor names the file.
- An anchor with an unknown ID gives a warning on stderr with the line, and adds no row.

**Tests:** `cmd/spekk/constraints_test.go`.
