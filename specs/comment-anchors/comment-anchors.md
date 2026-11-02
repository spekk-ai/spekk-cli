---
id: comment-anchors
created: 2026-11-02T21:30:00Z
priority: 1
---

# Comment Anchors - the Code That Enforces a Property Names It

## Problem

A property is compressed context only when an agent can find the code behind it. A comment anchor marks the line in the code that enforces a property: the guard, the database constraint, the lock. Other specs use the anchors: `constraints-command` lists the properties of a file, and `verify-command` marks a prose property `unconfirmed` when its anchored code changes.

An anchor that names no property is a false claim, so `spekk validate` rejects it.

## Anchor syntax

A line comment that starts with `#`, `//` or `--`, then `spekk:`, at least one space, and the ID of a property. Nothing follows the ID on the line:

```python
if new_balance < 0:  # spekk: balance-nonneg
    raise Overdraft()
```

```sql
CHECK (balance >= 0) -- spekk: balance-nonneg
```

## Decisions

- Comment anchors come first, and annotations come later (William, 2026-11-01).
- An anchor goes only at an enforcement point. It does not go on each line that touches the concept. The coach and builder prompts teach this (`property-prompts`). Code cannot check it.
- `spekk validate` fails on an anchor ID that does not exist.

## Resolved choices

- **Only the ID.** An anchor comment holds only `spekk:` and one ID. A comment with more words after `spekk:` is not an anchor. Two properties at one point need two comments. The reason: v1 projects have prose comments such as `# spekk: the binary is installed` (this repository has one in `scripts/assisted-by`). A rule that reads the first word as an ID makes such a project fail `spekk validate`.
- **Files that spekk scans.** Spekk scans the files that git tracks, and the untracked files that git does not ignore. It skips Markdown files (`.md`), because docs and specs show anchors as examples. It skips each path with a `testdata` directory segment, because test fixtures hold invalid anchors on purpose. It skips binary files (a NUL byte in the first 8000 bytes, the rule that git uses).
- **No git.** Outside a git work tree, spekk scans no files and `spekk validate` prints a warning that it did not check anchors.
- **Markers are not anchors.** A `spekk:` comment that is an evidence marker (`evidence-markers`) is not an anchor.
- **Index freshness.** The index stores the HEAD commit of its last build. A reader rebuilds the index when HEAD changed. An anchor that is not committed shows after the next `spekk index`. This is a known limit.

## Known limits

- The scanner reads lines, not the syntax of each language. A `#`, `//` or `--` inside a string that is followed by `spekk:` reads as an anchor.

## Assertions

See `assertions/` for what must be true.
