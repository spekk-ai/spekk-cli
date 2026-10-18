---
id: release-binaries-sha256-checked
parent: sandbox-spekk-binaries
created: 2026-10-18T05:40:00Z
priority: 1
status: done
branch: feat/sandbox-spekk-binaries
depends-on: deploy-installs-matching-cli
---

# A Binary That Does Not Match the Release sha256 Is Not Installed

## Success Criteria

- Before it downloads, the deploy reads the `digest` of each asset of the release from `https://api.github.com/repos/spekk-ai/spekk-cli/releases/tags/<tag>`, with no token.
- An asset with no `sha256:` digest stops the deploy before any copy, with an error that names the asset and the release.
- The install script on the machine checks each uploaded file with `sha256sum -c` before it replaces anything. A mismatch stops the script with `spekk: <file> does not match the release sha256`, the installed agent-client and CLI stay as they were, the service is not restarted, and the uploads are removed.
- Test: the install command, run in a local shell, refuses a file with a wrong sha256 and leaves the installed file unchanged.
- Live check: a copy changed on the way (a PATH shim for `scp`) is refused, and `status` shows the old versions.

**Tests:** `internal/sandbox/deploy_replace_test.go` (`TestInstallCommandRefusesAWrongSHA256`), `internal/sandbox/release_test.go` (`TestDownloadBinariesRefusesAnAssetWithNoDigest`, `TestReleaseDigests`). Live check (2026-10-18): with a PATH shim for `scp` that changes the CLI upload, `spekk sandbox deploy` failed with `spekk: spekk.staged does not match the release sha256`, no upload stayed in the login home, and `status` still showed the old CLI. Every release of spekk-cli on GitHub (56) has a digest for each asset, so the no-digest path has a test only.
