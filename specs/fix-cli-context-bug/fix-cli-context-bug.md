---
id: fix-cli-context-bug
created: 2026-01-28T21:22:00Z
priority: 1
---

# Fix CLI Context Bug

## Overview

The `spekk next` command is incorrectly reading specs from the spekk-cli installation directory instead of the current working directory. This breaks the fundamental design principle that the CLI should be context-aware and work on whatever project the user is currently in.

## Expected Behavior

When a user runs `spekk next` from any directory, the CLI should:
1. Look for a `specs/` directory in the current working directory
2. Parse and prioritize specs from that local `specs/` directory
3. Return the next priority assertion from the local project

## Current Bug

The CLI is reading from the spekk-cli's own `specs/` directory instead of `process.cwd() + '/specs/'`

## Impact

This makes the CLI unusable for external projects, defeating the entire purpose of having a global CLI tool for spec-driven development.