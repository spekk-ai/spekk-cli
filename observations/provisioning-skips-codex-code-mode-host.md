---
slug: provisioning-skips-codex-code-mode-host
type: code_spec_misalignment
severity: medium
status: resolved
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
At the time of the finding, the `codex-installed-on-every-sandbox` assertion was `done` and said cloud-init and prepare-machine install "the same pinned Codex CLI and code mode host for amd64 and arm64". Only the deploy repair path (`ensureCodexCLI`) installed `codex-code-mode-host`. Cloud-init, the CloudFormation user data and `prepare-machine.sh` installed the `codex` binary alone.

## Evidence at the time of the finding
- `specs/sandbox-codex-login/assertions/codex-installed-on-every-sandbox.md`, first success criterion: both provisioning paths install the CLI and the code mode host.
- `internal/sandbox/codex_cli.go:9`: the deploy health check requires `test -x /usr/local/bin/codex-code-mode-host`, and lines 22–26 download and install it.
- `internal/sandbox/cloud-init.yaml:179-192`: downloads and installs only `codex-$arch-unknown-linux-musl`; no code mode host.
- `internal/sandbox/aws-cloudformation.yaml:323-336`: same block, same omission.
- `scripts/prepare-machine.sh:108-121`: installs only `codex`, and its idempotence check looks at `codex --version` alone.
- A source search found code mode host installation only in `codex_cli.go`.

## Impact
Full cloud-init left a DigitalOcean or GCP machine without the host before deploy. Normal `spekk sandbox create` then ran deploy, which repaired the host before service restart. A machine prepared only with `prepare-machine.sh`, an AWS instance left at the CloudFormation user-data stage, or a template image builder before deploy exposed the missing host. Existing template images can still lack it until deploy repairs them. The original cloud-init generation and syntax checks did not exercise either binary.

## Resolution
Full cloud-init and its generated AWS user data install and check both pinned binaries. The final provisioning marker requires the completed Codex step. `prepare-machine.sh` and deploy check the host against the pinned release binary hash for the machine architecture. They repair a missing, damaged, or different-release host. An isolated `linux/amd64` run used real pinned binaries and a runnable host from another release. A focused Go test covers the deploy repair decision and failure. The assertion returned to `done` after those checks, failure checks, and the project checks passed.
