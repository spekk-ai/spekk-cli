---
id: code-spec-misalignment-spec-validator
created: 2026-01-28T22:47:00Z
type: code_spec_misalignment
severity: medium
affected_specs:
  - spec-parser
affected_files:
  - package.json
---

# Missing Spec-Validator Module Referenced in Package.json

## Issue Description
The `package.json` file references a spec-validator module in the non-existent `app/` directory, causing the `test:specs` script to fail. This indicates incomplete implementation or outdated package configuration.

## Evidence
From `package.json:18`:
```json
"test:specs": "node app/spec-validator/cli.js",
```

Attempting to run the script fails:
```
Error: Cannot find module '/Users/william/thinknimble/spekk-cli/app/spec-validator/cli.js'
```

Directory scan shows:
- No `app/` directory exists (all code is in `src/`)
- No `spec-validator` module exists anywhere in the codebase

## Impact
- The `npm test` command partially fails because `test:specs` cannot execute
- Spec validation functionality appears to be missing entirely
- CI/CD pipelines may be affected if they rely on this test script
- Confusion about whether spec validation is implemented or planned

## Recommendation
Either:
1. Remove the `test:specs` script if spec validation is not needed
2. Implement the spec-validator module in the correct location (`src/spec-validator/`)
3. Update the script to point to an existing validation tool if one exists