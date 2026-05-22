---
id: install-module-has-unit-tests
parent: skill-install-system
created: 2026-05-22T12:00:00Z
priority: 2
status: not_started
depends-on: install-default-scope-is-local
branch: feature/skill-install-system
---

# Install Module Has Unit Tests

## Description

The install/uninstall/list logic lives in `internal/install/` and has unit tests covering the behaviors asserted by the rest of this spec. HTTP fetches are tested against an `httptest.Server` (no real network calls in tests).

## Success Criteria

- Tests live under `internal/install/*_test.go`
- A test exercises default-scope local writes using `t.TempDir()` for both home and cwd
- A test exercises `--global` writes
- A test asserts conflict refusal without `--force` and overwrite with `--force`
- A test exercises `--source` filename derivation (explicit `<skill>` vs basename fallback vs unusable URL)
- A test exercises agent validation (unknown agent → error)
- A test uses `httptest.NewServer` and `SPEKK_SKILLS_RAW_BASE` to verify the constructed registry URL
- A test uses `httptest.NewServer` and `SPEKK_SKILLS_API_BASE` to verify `--list` parsing of the GitHub contents JSON
- A test exercises uninstall: removes the file when present, errors when absent, never touches outside the scope dir
- A test exercises the `--global` + `--local` mutual-exclusion error
- All tests pass under `go test ./internal/install/...`
- No test makes a real network call
