---
id: codex-installed-on-every-sandbox
parent: sandbox-codex-login
created: 2026-10-19T00:00:00Z
priority: 1
status: in_progress
locked-by: builder-cch2-1-1792538000
branch: observer/provisioning-skips-codex-code-mode-host
---

# Both provisioning paths install Codex

## Success Criteria

- Full cloud-init on DigitalOcean and GCP, generated AWS user data, and prepare-machine install the same pinned Codex CLI and code mode host for amd64 and arm64, alongside Claude Code. A newly built template image has both binaries from its full cloud-init build.
- Prepare-machine and deploy accept an installed pair only when the CLI reports the pinned release and the runnable code mode host matches the pinned release. A runnable host from a different release, a missing host, or a damaged host makes each path install the pinned pair.
- A failed repair returns an error. A failed binary download or extraction leaves no successful provisioning marker.

## Verification

- The real pinned CLI and host run in an isolated filesystem. A real host from a different release, a missing host, a damaged host, and a complete pinned pair have observed results on prepare and deploy. A focused Go test proves that a failed host check invokes repair and that repair failure returns an error. Cloud-init generation, syntax, build, vet, tests, and format gates pass.
