---
id: strictness-leaves-display-unchanged
parent: strictness-config
created: 2026-11-02T21:30:00Z
priority: 2
status: not_started
depends-on: strictness-sets-the-ci-result
---

# Strictness Does Not Change What Spekk Displays

## Description

Strictness decides whether CI fails. It does not hide a value or change a value. A project with no strictness sees `unchecked` as a project with strictness does.

## Success criteria

- For the same specs and the same verify results, `spekk status`, `spekk list` (all formats), `spekk show --json` and `spekk constraints` give the same output with and without a `[strict]` table.
- No command hides or renames `unchecked` or `unconfirmed` because of a strictness setting.
