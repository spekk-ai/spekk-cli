---
id: index-db-gitignored
parent: sqlite-index
created: 2026-07-12T22:00:00Z
priority: 1
status: done
depends-on: index-command-builds-db
---

# Index Database Files Are Added to `.gitignore`

## Description

The SQLite index is a local build artifact derived from the Markdown source files. Its files must not be committed to the repo. `spekk index` ensures `.spekk/index.db`, `.spekk/index.db-wal`, and `.spekk/index.db-shm` are gitignored.

## Success Criteria

- After running `spekk index` in a repo whose `.gitignore` does not yet mention `.spekk/`, the repo root `.gitignore` contains a line for `.spekk/index.db*` or `.spekk/`.
- If `.gitignore` already contains `.spekk/index.db`, `spekk index` adds the new pattern once and does not add a duplicate on later runs.
- `git status` in the test repo after an index rebuild does not show any `.spekk/` file as untracked.
- If no `.gitignore` exists at the repo root, `spekk index` creates one containing the new pattern.
