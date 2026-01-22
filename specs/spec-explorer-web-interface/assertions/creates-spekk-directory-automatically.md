---
id: creates-spekk-directory-automatically
parent: spec-explorer-web-interface
created: 2026-01-22T21:00:00Z
priority: 1
status: not_started
---

# Creates .spekk Directory Automatically

## Assertion

The command creates the `.spekk/` directory automatically if it doesn't exist.

## Success Criteria

- Running `spekk show` in a directory without `.spekk/` creates the directory
- No error occurs when `.spekk/` directory already exists
- Directory is created with appropriate permissions

## Test Plan

```bash
# Test creation
rm -rf .spekk
spekk show
test -d .spekk

# Test existing directory
spekk show  # Should not error
```