---
id: ci-runs-all-tests
parent: spec-parser
created: 2026-01-20T19:45:00Z
priority: 2
status: not_started
---

# CI Runs All Tests

## What Must Be True

A CI configuration must exist that automatically runs all tests (implementation and sidecar) on every push and pull request.

Test commands must work cross-platform on macOS, Linux, and Windows.

## CI Configuration Location

```
.github/
└── workflows/
    └── test.yml      # GitHub Actions workflow
```

## Expected Behavior

When code is pushed or a PR is opened:

1. **Setup environment** - Install Node.js and dependencies
2. **Run implementation tests** - Execute `npm run test:impl`
3. **Run sidecar tests** - Execute `npm run test:specs`
4. **Report results** - Show pass/fail status
5. **Fail CI if any tests fail** - Block PRs if tests don't pass

## Workflow Triggers

Run tests on:
- Push to any branch
- Pull request opened or updated
- Manual workflow dispatch (optional)

## Required Steps

**1. Checkout code**
```yaml
- uses: actions/checkout@v4
```

**2. Setup Node.js**
```yaml
- uses: actions/setup-node@v4
  with:
    node-version: '20'
```

**3. Install dependencies**
```yaml
- run: npm ci
```

**4. Run implementation tests**
```yaml
- name: Run implementation tests
  run: npm run test:impl
```

**5. Run sidecar tests**
```yaml
- name: Run sidecar validation tests
  run: npm run test:specs
```

## Success Reporting

CI should:
- ✅ Show green checkmark if all tests pass
- ❌ Show red X if any tests fail
- Display test output in CI logs
- Block PR merges if tests fail (branch protection)

## package.json Test Scripts

**Critical:** Test scripts must work cross-platform (macOS, Linux, Windows).

```json
{
  "scripts": {
    "test": "npm run test:impl && npm run test:specs",
    "test:impl": "node src/test-runner.js",
    "test:specs": "node src/spec-validator/cli.js"
  }
}
```

**Implementation:** Use a test runner script (`src/test-runner.js`) that uses the `glob` package to find test files. This works reliably across all platforms, unlike shell glob expansion which behaves differently on macOS, Linux, and Windows.

```javascript
// src/test-runner.js
import { glob } from 'glob';
import { spawn } from 'child_process';

async function runTests() {
  const testFiles = await glob('src/**/__tests__/**/*.test.js');
  const nodeTest = spawn('node', ['--test', ...testFiles], {
    stdio: 'inherit'
  });
  nodeTest.on('close', (code) => process.exit(code));
}
runTests();
```

This approach:
- ✅ Works consistently on macOS, Linux, and Windows
- ✅ Doesn't rely on shell glob expansion
- ✅ Explicitly lists all test files to Node.js test runner
- ✅ Uses the `glob` package (already in dependencies)

## Success Criteria

- ✅ `.github/workflows/test.yml` exists
- ✅ Workflow triggers on push and pull_request
- ✅ Workflow sets up Node.js environment
- ✅ Workflow installs npm dependencies
- ✅ Workflow runs `npm run test:impl` (implementation tests)
- ✅ Workflow runs `npm run test:specs` (sidecar tests)
- ✅ **app/test-runner.js exists and uses glob package**
- ✅ **Tests pass on Linux (GitHub Actions Ubuntu runner)**
- ✅ **Tests pass on macOS and Windows**
- ✅ Workflow fails if either test suite fails
- ✅ CI status visible on PRs and commits
- ✅ Test output visible in workflow logs
- ✅ Works on macOS, Linux, and Windows

**Tests:** Manual validation (trigger workflow and verify it runs)

## Example Workflow

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

## Notes

This ensures code quality by automatically validating:
- Implementation behavior (JS tests)
- Spec system integrity (sidecar tests)

Before any code is merged, both test suites must pass.

**Cross-platform compatibility:** The test runner script approach works reliably on all platforms without depending on shell-specific glob expansion behavior
