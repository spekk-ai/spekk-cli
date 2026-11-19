---
id: tests-pass-without-network
parent: tests-pass-on-any-machine
created: 2026-11-19T15:58:31Z
priority: 2
status: done
branch: feature/tests-pass-on-any-machine
---

# `go test ./...` Passes With No Network Access

## Success Criteria

- `go test ./... -count=1` passes on a machine where public hostnames (for example `github.com`) do not resolve and outbound connections fail.
- No test opens a connection to a host outside loopback. A test that needs a remote endpoint uses a local `httptest` server or a fixture.
- A test that can only run against a real remote service is skipped unless an explicit opt-in environment variable is set. The skip message names that variable.
- `internal/sandbox` passes with the network off, and finishes within CI's `-timeout 120s`. It does not wait for DNS or connect timeouts.

**Note:** "Passes" means pass or skip with a stated reason. A test that fails offline is a failure, even when it passes online.

**Tests:** internal/testenv/testenv_test.go (`TestBlockRemoteHostsRefusesAnythingOutsideLoopback`), `TestMain` in every test package (testenv.Main blocks non-loopback dials and DNS, and drops editor prompt programs such as GIT_ASKPASS that made internal/sandbox wait for a person), internal/sandbox/machine_test.go (`TestPrepareMachineWorkspaceRefusesGit` skips where realpath lacks -m). Verified with the full suite under a macOS sandbox profile that denies all non-loopback network.
