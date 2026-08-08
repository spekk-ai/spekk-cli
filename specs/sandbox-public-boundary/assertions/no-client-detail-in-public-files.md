---
id: no-client-detail-in-public-files
parent: sandbox-public-boundary
created: 2026-08-08T21:00:00Z
priority: 1
status: done
---

# No Client Detail Reaches the Public Tree, and CI Says So

## Description

The control-host half of this boundary was already stated. The client half was not, and it leaked: a coach prompt taught through a real advisory conversation including the client's commercial position, a release note quoted an internal remark about whether a named person had signed off, and one client's spec vocabulary had spread into five example files across docs, specs and tests.

None of it failed a build. Example content is exactly the material no compiler reads, so the boundary needs a check of its own or it holds only as long as everyone remembers it.

## Success Criteria

- `scripts/check-private-terms.sh` fails when any supplied term appears in a tracked file, and prints every file and line so the author can fix it without guessing.
- The terms are read from `SPEKK_PRIVATE_TERMS`, or from a gitignored `.private-terms` for local runs. **No term list is committed**: a denylist in a public repository publishes the names it exists to keep out.
- With no terms supplied the script exits 2. It never reports success for a tree it did not scan — a check that silently passes when unconfigured is worse than no check, because it is believed.
- Matching is literal and case-insensitive, and deliberately over-matches. A false positive costs one exclusion; a false negative costs a disclosure.
- The check runs in CI on every push (`.github/workflows/hygiene.yml`). It runs on `push` rather than `pull_request` because a fork's pull request receives no secrets, and the check would then fail every outside contribution for a reason the contributor cannot fix.
- Generated and vendored files are excluded by path, not by pattern, so an exclusion cannot silently widen.
