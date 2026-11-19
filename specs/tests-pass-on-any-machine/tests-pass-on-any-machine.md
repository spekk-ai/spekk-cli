---
id: tests-pass-on-any-machine
created: 2026-11-19T15:58:31Z
priority: 2
---

# The Test Suite Passes on Any Developer Machine

## Overview

`go test ./...` passes in CI (Linux, clean runner) but fails on an ordinary macOS developer machine. That machine has a global git identity, a real app config directory, BSD userland, and possibly no network. Each failure fails on `main`, so a contributor cannot tell their own breakage from the noise.

The failures we observed, and the cause of each:

- `internal/observer` identity tests (`TestResolvesIdentAgreesWithWhatCommitTreeAccepts`, `TestAnnounceMarksTheFindingWhenTheRepoHasNoIdentity`) unset the repository's `user.name` and `user.email`, but git still reads the developer's global config. So the "no identity" case has an identity.
- `cmd/spekk` app tests (`TestChangedRequestBodyKeepsConflict`, `TestLaunchRecoversLostResponseWithoutSecondPost`, and others) set `XDG_CONFIG_HOME`. On macOS, `os.UserConfigDir()` ignores that variable and returns `~/Library/Application Support`. The tests then read and write the developer's real request store, and entries left by earlier runs conflict with new ones.
- `TestBuilderLoopExitWhenDone` uses a fake `claude` script that runs `sed -i 's/…/…/' file`. BSD `sed` reads the expression as the backup suffix, so the script fails and the loop is killed at its timeout.
- `internal/sandbox` tests fail when github.com does not resolve.

CI runs only on `ubuntu-24.04`, so none of these failures shows up there.

## Assertions

- `tests-ignore-developer-git-config`: tests that run git produce the same result whatever global or system git config the developer has.
- `app-tests-never-touch-the-real-config-dir`: app-client tests use a per-test request store on every OS, and never read or write the developer's real config directory.
- `test-helper-scripts-are-portable`: shell scripts that tests write run under both BSD and GNU userland.
- `tests-pass-without-network`: `go test ./...` passes with no network access.
- `ci-runs-tests-on-macos`: CI runs the full suite on macOS as well as Linux.
