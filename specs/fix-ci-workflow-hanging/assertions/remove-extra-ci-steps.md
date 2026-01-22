---
id: remove-extra-ci-steps
parent: fix-ci-workflow-hanging
created: 2026-01-22T21:10:00Z
priority: 1
status: done
---

# Remove Extra CI Steps

## What Must Be True

The `.github/workflows/test.yml` file must be simplified to only include the steps specified in the original CI requirements, removing the extra steps that cause hanging.

### Steps to Remove

The following steps are **not** specified in `specs/spec-parser/assertions/ci-runs-all-tests.md` and must be removed:

1. **"Validate CLI commands"** step:
   ```yaml
   - name: Validate CLI commands
     run: |
       node bin/spekk.js --help
       node bin/spekk.js loop --help
       npm run next
   ```

2. **"Test CLI functionality"** step:
   ```yaml
   - name: Test CLI functionality
     run: |
       # Test that core commands work
       node bin/spekk.js coach --help || echo "Coach help test complete"
       node bin/spekk.js builder --help || echo "Builder help test complete"
       
       # Test parser integration - verify it outputs valid JSON
       echo "Testing parser JSON output..."
       node src/parser/cli.js > /tmp/parser_output.json
       # ... rest of JSON validation
   ```

### Required Final Workflow

The workflow must contain **only** these steps:

```yaml
name: Tests

on:
  push:
    branches: ['**']
  pull_request:
    branches: ['**']

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Install dependencies
        run: npm ci

      - name: Run implementation tests
        run: npm run test:impl

      - name: Run sidecar validation tests
        run: npm run test:specs
```

### Why These Steps Must Be Removed

1. **Interactive Commands:** `coach --help` and `builder --help` start interactive Claude Code sessions that wait for user input
2. **Not Specified:** These steps aren't required by any existing spec
3. **Causing Failures:** They cause CI to hang indefinitely 
4. **Scope Creep:** They go beyond the original CI specification

### Implementation Notes

The original CI spec (`ci-runs-all-tests.md`) was already marked as `status: done`, meaning the basic CI functionality was complete. The extra steps were added later and are causing the current issues.

## Success Criteria

- ✅ `.github/workflows/test.yml` contains exactly 5 steps (checkout, setup-node, install, test:impl, test:specs)
- ✅ No "Validate CLI commands" step exists
- ✅ No "Test CLI functionality" step exists  
- ✅ No interactive commands (`coach`, `builder`) in workflow
- ✅ CI completes in under 2 minutes without hanging
- ✅ Implementation tests pass (`npm run test:impl` returns 0)
- ✅ Sidecar tests pass (`npm run test:specs` returns 0)