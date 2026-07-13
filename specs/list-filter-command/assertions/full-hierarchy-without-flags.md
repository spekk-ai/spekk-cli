---
id: full-hierarchy-without-flags
parent: list-filter-command
created: 2026-07-12T20:30:00Z
priority: 1
status: done
branch: feat/list-filter-by-status
depends-on: list-subcommand-registered
---

# `spekk list` Without Flags Returns Full Hierarchy

## Description

With no filtering flags, `spekk list` outputs the same JSON as `spekk next --all`:
the complete hierarchy of all specs and assertions.

## Success Criteria

- `spekk list` output is valid JSON with `"type": "hierarchy"`.
- `spekk list` and `spekk next --all` produce identical output (same specs,
  same assertions, same sort order) when run in the same directory.
- The output includes the `"observations": []` field.
- `spekk list --specs-dir <path>` works the same way `spekk next --specs-dir <path>`
  does (reads specs from an explicit directory rather than git root).
