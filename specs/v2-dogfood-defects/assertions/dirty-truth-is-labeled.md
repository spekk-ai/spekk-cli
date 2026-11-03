---
id: dirty-truth-is-labeled
parent: v2-dogfood-defects
created: 2026-11-03T06:49:00Z
priority: 1
status: done
branch: fix/v2-dogfood-defects
---

# Dirty Truth Is Labeled in Every Reader

## Success criteria

- After a verify run with uncommitted changes, `spekk list` and `spekk constraints` print the dirty-run note on stderr in text modes.
- `spekk list --json`, `spekk show --json`, and `spekk constraints --json` include a true `dirty` value with their truth output.
- A run on a clean tree gives a false `dirty` value and no text note.
