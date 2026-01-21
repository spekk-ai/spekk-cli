---
id: spekk-cli
created: 2026-01-21T19:00:00Z
priority: 2
status: not_started
---

# Spekk CLI

## Overview

The spekk-cli command-line interface provides core spec-driven development commands for parsing specs, launching agents, and managing workflows across any spec-driven project.

## Core Commands

### Parser Commands
- `spekk` (default) - Parse specs in current directory and return next priority item
- `spekk next` - Alias for default parser behavior
- `spekk status` - Show status overview of all specs and assertions

### Agent Commands  
- `spekk coach` - Launch coach loop (interactive agent sessions) for current directory
- `spekk builder` - Launch builder loop (automated implementation) for current directory

### Utility Commands
- `spekk --version` - Show CLI version
- `spekk --help` - Show command help

## Status Overview Command

The `spekk status` command provides a comprehensive view of spec progress:

```bash
$ spekk status

📋 Spec Status Overview

✅ semantic-versioning (3/3 assertions complete)
🚧 spekk-cli-extraction (1/7 assertions complete)
   ✅ project-structure-exists
   📋 specs-moved-to-cli  
   📋 cli-parses-external-specs
   📋 agent-loops-work-externally
   📋 local-parser-removed
   📋 cli-project-is-spec-driven
   📋 current-project-uses-spekk-cli
⏸️  other-spec (0/2 assertions complete)

Next Priority: specs-moved-to-cli (spekk-cli-extraction)
```

## Design Principles

**Context Awareness:** CLI operates on current working directory's specs/ folder
**Global Installation:** Available as `spekk` command anywhere on system  
**Self-Contained:** No dependencies on local project structure
**Fast Execution:** All operations complete in < 100ms
**Cross-Platform:** Works on macOS, Linux, Windows