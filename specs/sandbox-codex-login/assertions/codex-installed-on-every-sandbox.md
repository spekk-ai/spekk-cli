---
id: codex-installed-on-every-sandbox
parent: sandbox-codex-login
created: 2026-10-19T00:00:00Z
priority: 1
status: done
branch: sc1-codex-sandboxes
---

# Both provisioning paths install Codex

## Success Criteria

- Cloud-init and prepare-machine install the same pinned Codex CLI for amd64 and arm64, alongside Claude Code.
- Deploy checks the installed version and repairs a missing or different version before service restart.
- Failed download or extraction stops provisioning.

## Verification

- Go tests verify deploy ordering and its version check. Cloud-init generation and syntax gates pass.
