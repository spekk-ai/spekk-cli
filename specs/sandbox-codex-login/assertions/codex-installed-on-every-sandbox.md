---
id: codex-installed-on-every-sandbox
parent: sandbox-codex-login
created: 2026-10-19T00:00:00Z
priority: 1
status: done
branch: observer/provisioning-skips-codex-code-mode-host
---

# Both provisioning paths install Codex

## Success Criteria

- Full cloud-init on DigitalOcean and GCP, generated AWS user data, and prepare-machine install the same pinned Codex CLI and code mode host for amd64 and arm64, alongside Claude Code. A newly built template image has both binaries from its full cloud-init build.
- Deploy checks the installed version and code mode host and repairs a missing or different version before service restart.
- A missing or unusable host makes prepare-machine repair the install. A failed binary download or extraction leaves no successful provisioning marker.

## Verification

- The real shell paths install both binaries in an isolated filesystem. Missing-host repair, a complete install, failed host download, and corrupt archive have observed results. Cloud-init generation, syntax, build, vet, tests, and format gates pass.
