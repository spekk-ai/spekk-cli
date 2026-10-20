---
slug: provisioning-skips-codex-code-mode-host
type: code_spec_misalignment
severity: medium
status: open
created: 2026-10-20T12:00:00Z
announced: 2026-10-20T19:01:34Z
affected:
  - specs/sandbox-codex-login/assertions/codex-installed-on-every-sandbox.md
  - internal/sandbox/cloud-init.yaml
  - internal/sandbox/aws-cloudformation.yaml
  - scripts/prepare-machine.sh
  - internal/sandbox/codex_cli.go
---

# Provisioning paths install Codex without the code mode host

## Issue Description
The `codex-installed-on-every-sandbox` assertion is `done` and says cloud-init and prepare-machine install "the same pinned Codex CLI and code mode host for amd64 and arm64". Only the deploy repair path (`ensureCodexCLI`) installs `codex-code-mode-host`. Cloud-init, the CloudFormation user data and `prepare-machine.sh` install the `codex` binary alone, so a freshly provisioned sandbox lacks the host until a deploy repairs it.

## Evidence
- `specs/sandbox-codex-login/assertions/codex-installed-on-every-sandbox.md`, first success criterion: both provisioning paths install the CLI and the code mode host.
- `internal/sandbox/codex_cli.go:9`: the deploy health check requires `test -x /usr/local/bin/codex-code-mode-host`, and lines 22–26 download and install it.
- `internal/sandbox/cloud-init.yaml:179-192`: downloads and installs only `codex-$arch-unknown-linux-musl`; no code mode host.
- `internal/sandbox/aws-cloudformation.yaml:323-336`: same block, same omission.
- `scripts/prepare-machine.sh:108-121`: installs only `codex`, and its idempotence check looks at `codex --version` alone.
- `grep -rn code-mode internal scripts` finds hits only in `codex_cli.go`.

## Impact
A sandbox that is provisioned but not yet deployed does not match the spec. Deploy then fails its health check every time and has to download Codex again. Any Codex feature that needs the host fails on machines prepared only by `prepare-machine.sh`. The assertion's own verification (cloud-init generation and syntax gates) cannot catch this, which explains how it was marked done.

## Recommendation
Re-queue the assertion (status flipped to `failed` in this branch). In all three provisioning scripts, the builder should install `codex-code-mode-host` from the same pinned release. `prepare-machine.sh` should also check for the host before it skips the install. A test that checks each provisioning script for both binaries would stop this from coming back.