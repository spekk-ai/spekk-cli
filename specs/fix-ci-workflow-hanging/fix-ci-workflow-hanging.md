---
id: fix-ci-workflow-hanging
created: 2026-01-22T21:10:00Z
priority: 1
---

# Fix CI Workflow Hanging

## Problem

CI is hanging indefinitely on the "Test CLI functionality" step because it tries to run interactive coach and builder commands that wait for user input, which never comes in CI environment.

## What Must Be True

The GitHub Actions CI workflow must only include the steps that are specified in the original CI requirements and must complete successfully without hanging.

### Current CI Issues

The workflow includes extra steps not specified in `specs/spec-parser/assertions/ci-runs-all-tests.md`:

1. **"Validate CLI commands"** - Not required by spec
2. **"Test CLI functionality"** - Hangs on interactive commands

These steps try to run:
```bash
node bin/spekk.js coach --help  # Hangs - starts interactive session
node bin/spekk.js builder --help # Hangs - starts interactive session
```

### Required CI Steps (Per Original Spec)

The CI workflow should **only** include what's specified in `ci-runs-all-tests.md`:

1. ✅ Setup environment (checkout, Node.js, dependencies)
2. ✅ Run implementation tests (`npm run test:impl`)
3. ✅ Run sidecar tests (`npm run test:specs`)
4. ✅ Report pass/fail status

### CI Workflow Should Be

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

## Success Criteria

- ✅ CI workflow completes without hanging
- ✅ CI workflow only includes steps specified in `ci-runs-all-tests.md`
- ✅ Implementation tests pass in CI
- ✅ Sidecar validation tests pass in CI
- ✅ No interactive commands in CI workflow
- ✅ CI run time is reasonable (under 5 minutes)
- ✅ Workflow fails only if tests fail, not due to hanging