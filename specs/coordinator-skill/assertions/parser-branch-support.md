---
id: parser-branch-support
parent: coordinator-skill
created: 2026-02-25T00:00:00Z
priority: 2
status: done
---

# Parser Validates and Parses branch Field

Spec parser reads and validates the `branch` field in assertion frontmatter.

## What Must Be True

### Field Parsing
- [ ] Parser extracts `branch` field from YAML frontmatter
- [ ] Field value is stored in parsed assertion object
- [ ] Omitted `branch` field defaults to `main`

### Field Validation
- [ ] `branch` must be a string if present
- [ ] Value must be a valid git branch name:
  - No spaces or control characters
  - Only letters, numbers, `/`, `-`, `_`
  - Cannot start or end with `/`
  - Cannot end with `.`
  - Cannot contain `..`, `@{`, `\`, `~`, `^`, `:`, `?`, `*`, `[`
- [ ] Invalid values produce clear error messages
- [ ] Non-standard patterns (not `main`, `feature/*`, `bugfix/*`, `hotfix/*`) produce warnings but don't fail

### Error Handling
- [ ] Type errors identify the file and field
- [ ] Format errors explain what's invalid
- [ ] Warnings suggest standard patterns
- [ ] All error messages are actionable

## Validation Points

- Parser successfully reads `branch` field
- Invalid characters are rejected
- Invalid formats are rejected  
- Non-standard patterns show warnings
- Omitted field defaults to `main`
- Error messages guide correction
