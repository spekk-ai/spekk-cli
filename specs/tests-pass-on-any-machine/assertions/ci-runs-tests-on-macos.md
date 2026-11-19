---
id: ci-runs-tests-on-macos
parent: tests-pass-on-any-machine
created: 2026-11-19T15:58:31Z
priority: 3
status: in_progress
branch: feature/tests-pass-on-any-machine
---

# CI Runs the Full Test Suite on macOS as Well as Linux

## Success Criteria

- `.github/workflows/test.yml` runs `go test ./... -count=1` on a macOS runner and on `ubuntu-24.04`. A failure on either fails the workflow.
- The macOS job runs with a global git identity configured, so a test that leaks the developer's git config fails in CI.
- Both jobs use the same Go version and the same test command.
- `publish.yml` still gates the release on passing tests.

**Tests:** internal/testenv/testenv_test.go (`TestCIRunsTheSuiteOnLinuxAndMacOS`); the macOS and Linux jobs of `.github/workflows/test.yml` on the PR
