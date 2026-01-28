---
id: robust-error-handling
created: 2026-01-28T21:35:00Z
priority: 1
status: not_started
---

# Robust Error Handling

## Overview

The CLI should gracefully handle malformed files and parsing errors instead of completely failing. Users should get warnings but the system should continue to function.

## Current Problem

The CLI completely fails and exits when it encounters malformed observation files or other parsing errors, making the entire system unusable.

## Required Behavior

- **Warn but continue** - Log warnings for malformed files
- **Skip invalid files** - Ignore files that can't be parsed
- **Never crash the CLI** - Always return valid JSON or continue processing
- **Provide helpful messages** - Tell users which files have issues