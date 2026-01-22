---
id: spekk-show-command-exists
parent: spec-explorer-web-interface
created: 2026-01-22T21:00:00Z
priority: 1
status: done
---

# Spekk Show Command Exists

## Assertion

The `spekk show` command is recognized and executable from the CLI.

## Success Criteria

- Running `spekk show` does not produce "command not found" or similar errors
- Command appears in `spekk --help` output
- Command has appropriate help text describing its functionality

**Tests:** src/__tests__/show-command.test.js

## Test Plan

```bash
spekk show --help
spekk --help | grep "show"
```