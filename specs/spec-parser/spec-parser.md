---
id: spec-parser
created: 2026-01-20T15:35:00Z
priority: 1
status: in_progress
---

# Spec Parser

## Overview

The spec parser is the foundation of the Spekk system - it reads specification files and identifies what to work on next. This is the "factory builder" that enables spec-driven development.

## Spec File Structure

### Folder Organization
Each spec is a folder containing:
- Main spec file: `{spec-id}/{spec-id}.md`
- Assertions subfolder: `{spec-id}/assertions/`
- Assertion files: `{spec-id}/assertions/{assertion-id}.md`

Example:
```
specs/
├── spec-parser/
│   ├── spec-parser.md
│   └── assertions/
│       ├── parses-frontmatter.md
│       ├── validates-required-fields.md
│       └── ...
```

### File Format
Both specs and assertions are Markdown files with YAML frontmatter.

**Spec file frontmatter:**
```yaml
---
id: spec-parser              # Unique identifier (kebab-case)
created: 2026-01-20T15:35:00Z  # ISO 8601 timestamp (immutable)
priority: 1                   # 1 (highest) | 2 (medium) | 3 (lowest)
status: not_started          # not_started | in_progress | done
---
```

**Assertion file frontmatter:**
```yaml
---
id: parses-frontmatter       # Unique within parent spec
parent: spec-parser          # Parent spec id
created: 2026-01-20T16:00:00Z
priority: 1                   # 1 (highest) | 2 (medium) | 3 (lowest)
status: not_started
---
```

### Priority Levels

Only three priority levels are allowed:
- **1** - Highest priority (critical, blocking)
- **2** - Medium priority (important, but not blocking)
- **3** - Lowest priority (nice to have, future work)

This constraint keeps prioritization simple and forces clear decisions about importance.

## What the Parser Does

### 1. Read and Parse
- Recursively scans `specs/` directory for spec folders
- Reads all `.md` files (specs and assertions)
- Extracts and validates YAML frontmatter
- Builds hierarchical structure: specs → assertions

### 2. Validate Format
- Ensures required fields exist (`id`, `created`, `priority`)
- Validates field types and formats
- Checks for duplicate IDs
- Validates folder structure matches declared format
- Reports errors for malformed specs

### 3. Identify Next Work Item
- Filters to incomplete items (status != "done")
- Sorts by priority (1 is highest, then 2, then 3)
- **Tie-breaking**: When multiple items have same priority, select oldest by `created` timestamp
- Returns the single highest-priority incomplete assertion
- If all assertions done for a spec, returns next spec

### 4. Output Format
Machine-readable JSON output:
```json
{
  "type": "assertion",
  "id": "parses-frontmatter",
  "parent": "spec-parser",
  "file": "specs/spec-parser/assertions/parses-frontmatter.md",
  "priority": 0,
  "status": "not_started",
  "title": "Parser Must Extract YAML Frontmatter"
}
```

## Status Aggregation

Specs derive their status from assertions:
- Spec is `done` only when ALL assertions are `done`
- If ANY assertion is `in_progress`, spec shows as `in_progress`
- If ANY assertion is `not_started`, spec shows as `not_started`

Status values:
- `not_started` - Not yet implemented (default)
- `in_progress` - Currently being worked on
- `done` - Implemented and validated
- `draft` - Placeholder/planning stage (excluded from work queue)

## Implementation

The parser is a **Node.js module** in `src/parser/`.

**Key Design Principles:**
- **Standalone**: Can be run independently as CLI tool
- **Language agnostic**: Outputs JSON that any language can consume
- **Fast**: Executes in < 100ms even with many specs
- **Self-contained**: Minimal dependencies, no database required

**Current Implementation:**
- Language: Node.js + ES modules
- Location: `src/parser/`
- Dependencies: Built-in YAML frontmatter parsing (no external deps needed)
- Interface: CLI that outputs JSON to stdout

**Usage:**
```bash
# From project root
npm run next

# Or directly
node src/parser/cli.js

# Returns JSON to stdout
```

## Why This Exists

The parser enables spec-driven development. Without it:
- Agents/developers manually scan files to find next priority
- No validation that specs follow the declared format
- No automation of the "what's next?" workflow

With the parser:
- Ralph loop can automate work: `parser → read assertion → work → mark done → commit`
- CI can verify specs are well-formed
- Dashboards can show status and priorities
- The system becomes self-hosting

This is infrastructure for the infrastructure.

## Assertions

See `assertions/` subfolder for detailed validation criteria and behaviors that must be true for this spec to be considered complete.
