---
id: spec-parser-works-externally
parent: cli-prompt-resolution
created: 2026-01-28T20:15:00Z
priority: 1
status: not_started
---

# Spec Parser Works Externally

## Assertion

The spec parser (default `spekk` command) works when run from any directory, not just from within the spekk-cli installation directory.

## Success Criteria

- Running `spekk` (no arguments) from external directory successfully finds next assertion
- Running `spekk --all` from external directory lists all specs and assertions
- Parser correctly locates `specs/` directory from spekk-cli installation, not current directory
- No "directory not found" or "no specs found" errors when running from external directories
- Parser output is identical whether run from spekk-cli directory or external directory

## Context

Currently, the spec parser looks for `specs/` relative to the current working directory. When run from external directories (like `~/thinknimble/vuenome`), it can't find the specs and fails.

This is the same path resolution issue that affected agent prompts. The parser needs to resolve the specs directory relative to the spekk-cli installation, not the user's working directory.

## Implementation Requirements

The parser should:
- Use `__dirname` or similar to locate spekk-cli installation directory
- Read specs from `{installation}/specs/` not `{cwd}/specs/`
- Maintain all existing functionality (finding next assertion, listing all specs)
- Work consistently from any directory