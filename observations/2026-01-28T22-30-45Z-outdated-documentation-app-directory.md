---
id: outdated-documentation-app-directory
created: 2026-01-28T22:30:45Z
type: outdated_specs
severity: low
affected_specs:
  - spec-parser
affected_files:
  - CLAUDE.md
---

# Outdated Documentation: Non-existent app/ Directory

## Issue Description
The project's CLAUDE.md file references an `app/` directory that doesn't exist in the current implementation. The documentation states that `app/` contains "Core application logic for parser, coach, and builder", but all application logic is actually in the `src/` directory.

## Evidence
From `CLAUDE.md`:
```
## Project Structure

- `app/` - Core application logic for parser, coach, and builder
- `bin/` - CLI entry points 
- `specs/` - Specification files organized by feature
```

Actual project structure:
```
- `src/` - Core application logic (parser, coach, builder, etc.)
- `bin/` - CLI entry points
- `specs/` - Specification files organized by feature
```

## Impact
- Creates confusion for developers trying to understand the project structure
- May lead to incorrect assumptions about where to find or place code
- Indicates documentation has not been updated after a refactoring

## Recommendation
Update CLAUDE.md to reflect the actual project structure, changing `app/` to `src/` in the Project Structure section.