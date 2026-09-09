---
id: announce-uses-live-claim-rule
parent: only-a-live-claim-covers
created: 2026-09-09T12:00:00Z
priority: 1
status: done
depends-on: covering-needs-the-owning-branch
---

# Announce Uses the Same Active-Finding Rule as Deduplication and Digest

## Description

Announce, deduplication, and digest agree on whether an observation is an active finding. The observation must be on its own `observer/<slug>` branch, and its slug must be absent from main. This fixes issue #211.

## Success Criteria

- All three operations use one shared rule for branch ownership and presence on main.
- Local and remote-tracking refs for the observation's own branch pass this rule.
- All three operations recognize main and master by the complete branch name. Remote branches `observer/main` and `observer/master` remain observation branches.
- A copy on a renamed branch or another finding's branch fails the rule.
- A copy on another branch cannot displace an eligible copy of the same observation on its own branch.
- Presence on main ends the claim even if its own branch remains visible and its status is still `open`.
- Announce keeps its status, announcement-marker, severity, evidence, origin-visibility, ordering, and batch rules. Digest keeps its status filter, ordering, and limit. Deduplication keeps its type-and-slug match.

## Verification

`internal/observer/announce_test.go` reads Git fixtures through the union and the SQLite index to check agreement for local, remote, renamed, and merged findings, including slugs `main` and `master`. It also checks that an inherited copy cannot displace the copy on the observation's own branch. The observer and observation package tests cover the remaining selection rules.
