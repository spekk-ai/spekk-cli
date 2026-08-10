---
id: index-db-gitignored
parent: sqlite-index
created: 2026-07-12T22:00:00Z
priority: 1
status: done
depends-on: index-command-builds-db
---

# `.spekk/index.db` Is Added to `.gitignore`

## Description

The SQLite index is a local build artifact derived from the Markdown source
files. It must not be committed to the repo. `spekk index` ensures
`.spekk/index.db` (or the entire `.spekk/` directory) is gitignored.

## Success Criteria

- After running `spekk index` in a repo whose `.gitignore` does not yet
  mention `.spekk/`, the repo root `.gitignore` contains a line for
  `.spekk/index.db` or `.spekk/`.
- If `.gitignore` already contains the entry, `spekk index` does not add a
  duplicate.
- `git status` in the test repo after `spekk index` does not show
  `.spekk/index.db` as an untracked file (confirming gitignore works).
- If no `.gitignore` exists at the repo root, `spekk index` creates one
  containing the entry.
