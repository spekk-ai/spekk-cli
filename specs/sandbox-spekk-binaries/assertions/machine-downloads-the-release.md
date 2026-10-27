---
id: machine-downloads-the-release
parent: sandbox-spekk-binaries
created: 2026-10-23T00:00:00Z
priority: 1
status: done
branch: feat/machine-downloads-release
depends-on: release-binaries-sha256-checked
---

# The Machine Downloads the Release Binaries Itself

## Problem

The deploy step downloaded both binaries (about 40 MB) to the operator's computer and copied them to the machine with one `scp`. On a network that corrupts large transfers, ssh logs `message authentication code incorrect` and drops the connection, so `create` stops at `copying binary` every time, while every small step before it passes.

## Success Criteria

- The deploy step first runs one command over SSH, as the login user, that downloads `sandbox-linux-<arch>` and `spekk-linux-<arch>` from `https://github.com/spekk-ai/spekk-cli/releases/download/<tag>/` into the staged paths in the login home, with `curl`, else `wget`. No binary crosses the operator's connection.
- That command checks each file with `sha256sum -c` against the digest that the release publishes. A failed download or a mismatch removes both staged files and fails the command.
- The download has a time limit: `curl` stops an attempt when the transfer is below 1024 bytes per second for 30 seconds or runs longer than 300 seconds (`--speed-limit 1024 --speed-time 30 --max-time 300`). These limits apply to each attempt, and `--retry 2` gives 3 attempts, so a slow `curl` download can take approximately 15 minutes for each file (approximately 30 minutes for the two files) before the command fails. `wget` runs under `timeout 300` with `-T 10 --tries=2`, so its limit is 300 seconds for each file, for all tries. A stalled download thus fails the command, and the fallback below runs.
- When the command fails (no `curl` or `wget`, no route to GitHub, a mismatch), the deploy prints `spekk: the machine could not download release <tag> (<cause>); copying it from this computer`, downloads the binaries to the operator, and copies them with `scp`, as before.
- The operator downloads the binaries only for that fallback.
- The install script checks the staged files against the release sha256 either way, as `release-binaries-sha256-checked` says.
- Test: a deploy whose download command succeeds runs no `scp`; one whose download command fails copies with `scp` and then installs.
- Test: the download command, run in a local shell with a stub `curl`, leaves both staged files when they match, and leaves neither when one does not.

**Tests:** `internal/sandbox/ssh_user_test.go` (`TestDeployStagesAndInstallsForEachLogin`), `internal/sandbox/deploy_replace_test.go` (`TestFetchCommandChecksTheDownloads`), `internal/sandbox/release_test.go` (`TestDownloadBinariesSelectsArchAssets`, `TestSelectArchRefusesAnAssetWithNoDigest`).
