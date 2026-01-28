---
id: spec-conflict-parser-directory
created: 2026-01-28T22:30:15Z
type: spec_conflicts
severity: high
affected_specs:
  - cli-prompt-resolution
  - fix-cli-context-bug
affected_files:
  - src/parser/index.js
  - src/parser/__tests__/external-directory.test.js
  - specs/cli-prompt-resolution/assertions/spec-parser-works-externally.md
  - specs/fix-cli-context-bug/assertions/cli-reads-local-specs.md
---

# Conflicting Requirements: Parser Directory Resolution

## Issue Description
Two specifications have contradictory requirements about where the spec parser should read spec files from:

1. **cli-prompt-resolution/spec-parser-works-externally** requires the parser to read specs from the spekk-cli **installation directory**
2. **fix-cli-context-bug/cli-reads-local-specs** requires the parser to read specs from the **current working directory**

## Evidence
From `specs/cli-prompt-resolution/assertions/spec-parser-works-externally.md`:
```
When the spekk CLI is installed globally (npm install -g @thinknimble/spekk-cli), 
it needs to read spec files from its own installation directory, not the user's 
current working directory.
```

From `specs/fix-cli-context-bug/assertions/cli-reads-local-specs.md`:
```
The spekk CLI should be able to read and parse spec files from the user's 
current working directory, not from the spekk-cli installation directory.
```

Current implementation in `src/parser/index.js:304`:
```javascript
const specsPath = path.join(process.cwd(), 'specs');
```

Test failures in `src/parser/__tests__/external-directory.test.js`:
- ✗ parses specs from installation directory when run externally
- ✗ finds next assertion from external specs
- ✗ excludes draft specs when finding next assertion

## Impact
- The parser cannot satisfy both requirements simultaneously
- Tests are failing for the `spec-parser-works-externally` assertion
- The assertion is marked as "done" but is not actually implemented correctly
- This creates confusion about which behavior is correct

## Recommendation
1. Review both specs to determine the intended behavior
2. Either:
   - Consolidate the specs with a clear requirement
   - Update one spec to align with the chosen approach
   - Implement a flag/option to support both behaviors
3. Update assertion status to reflect actual implementation state