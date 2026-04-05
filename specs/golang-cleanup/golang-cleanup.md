---
id: golang-cleanup
created: 2026-04-05T12:32:00Z
priority: 3
---

# Go Migration Cleanup

Remove all Node.js code, dependencies, and build infrastructure after the full Go migration is complete.

This is the final phase — only executed after every module has been ported and validated in Go.

## Strategy

- Delete all JS source files and tests
- Remove package.json, node_modules, npm scripts
- Update CI/CD to build Go binary
- Update installation instructions for single binary distribution
- Update CLAUDE.md and spec prompts to reference Go commands
