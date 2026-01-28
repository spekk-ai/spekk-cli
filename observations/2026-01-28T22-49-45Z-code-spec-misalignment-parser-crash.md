---
id: code-spec-misalignment-parser-crash
created: 2026-01-28T22:49:45Z
type: code_spec_misalignment
severity: high
affected_specs:
  - observer-agent
  - spec-parser
affected_files:
  - src/parser/index.js
  - observations/2026-01-28T22-30-45Z-outdated-documentation-app-directory.md
---

# Parser Crashes on Valid Observation Files

## Issue Description
The spec parser crashes when processing observation files that have empty arrays for the `affected_specs` field, even though this is valid according to the observation format specification. This prevents the parser from running successfully, blocking core CLI functionality.

## Evidence
Running `npm run next` produces this error:
```
Error: Field 'affected_specs' must be an array in observations/2026-01-28T22-30-45Z-outdated-documentation-app-directory.md
```

The observation file that triggers the crash has valid YAML frontmatter:
```yaml
affected_specs: []
```

This format is consistent with the observation spec which allows for observations that don't directly affect specific specs (like outdated documentation).

## Impact
- The parser cannot run when any observation has an empty `affected_specs` array
- Core CLI commands like `npm run next` fail completely
- This contradicts the "done" status of parser-related specs
- Observer agent cannot create certain types of observations without breaking the system
- Blocks the entire spec-driven development workflow

## Recommendation
1. **Immediate fix:** Update the parser validation to accept empty arrays for `affected_specs`
2. **Root cause:** The parser may be checking for array length > 0 instead of just array type
3. **Testing:** Add test cases for observations with empty arrays to prevent regression
4. **Status update:** Consider marking parser specs as "in_progress" until this critical bug is fixed