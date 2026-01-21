---
id: sidecar-test-runner
parent: spec-parser
created: 2026-01-20T19:35:00Z
priority: 1
status: done
---

# Sidecar Test Runner Exists

## What Must Be True

A test runner must exist that discovers and runs all sidecar bash tests (`*.test.sh`) located alongside assertions in the spec tree.

## Test Runner Location

```
app/
└── spec-validator/
    ├── run-sidecar-tests.js    # Test runner implementation
    └── cli.js                   # CLI entry point
```

## Expected Behavior

When running:
```bash
npm run test:specs
```

The runner must:
1. Find all `specs/**/assertions/*.test.sh` files
2. Execute each bash script
3. Collect results (exit code 0 = pass, non-zero = fail)
4. Report only failures (silent on success)
5. Exit with code 0 if all pass, non-zero if any fail

## Output Format

**All tests pass:**
```
✅ All spec validation tests passed (12 tests)
```

**Some tests fail:**
```
❌ Failed: specs/builder-agent/assertions/prompt-exists.test.sh
   Builder prompt missing at specs/builder-agent/builder-agent.prompt.md

❌ Failed: specs/spec-parser/assertions/app-directory-structure.test.sh
   scripts/ directory still exists

Failed: 2 of 12 tests
```

## Implementation Requirements

**Discovery:**
- Use glob pattern: `specs/**/assertions/*.test.sh`
- Skip if no test files found (not an error)

**Execution:**
- Run each test with `bash <test-file>`
- Capture stdout/stderr
- Check exit code

**Reporting:**
- Only show output for failed tests
- Show test file path
- Show error message from test
- Count total and failed tests

**Exit behavior:**
- Exit 0: All tests passed
- Exit 1: One or more tests failed
- Exit 0: No tests found (not an error)

## package.json Script

```json
{
  "scripts": {
    "test": "npm run test:impl && npm run test:specs",
    "test:impl": "node --test 'app/**/__tests__/**/*.test.js'",
    "test:specs": "node app/spec-validator/cli.js"
  }
}
```

This separates:
- `npm run test:impl` - Implementation tests (JS)
- `npm run test:specs` - Sidecar validation tests (bash)
- `npm test` - Runs both

## Example Sidecar Test

```bash
#!/usr/bin/env bash
# specs/builder-agent/assertions/prompt-exists.test.sh

PROMPT_FILE="specs/builder-agent/builder-agent.prompt.md"

[ -f "$PROMPT_FILE" ] || {
  echo "❌ Builder prompt missing at $PROMPT_FILE"
  exit 1
}

grep -q "Work on ONE assertion" "$PROMPT_FILE" || {
  echo "❌ Builder prompt missing 'Work on ONE assertion' instruction"
  exit 1
}

# Silent on success
exit 0
```

## Success Criteria

- ✅ `app/spec-validator/` directory exists
- ✅ Test runner discovers all `*.test.sh` files in specs tree
- ✅ Runner executes bash scripts and captures exit codes
- ✅ Runner only reports failures (silent on success)
- ✅ Runner exits 0 if all pass, non-zero if any fail
- ✅ `npm run test:specs` command exists
- ✅ `npm test` runs both implementation and sidecar tests
- ✅ Works with zero test files (not an error)
- ✅ Clear error messages show which tests failed

**Tests:** `app/spec-validator/__tests__/run-sidecar-tests.test.js`

Test should validate:
- Runner discovers test files correctly
- Runner executes bash scripts
- Runner captures and reports failures
- Runner exits with correct code
