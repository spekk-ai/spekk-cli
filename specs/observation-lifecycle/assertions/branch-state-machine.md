---
id: branch-state-machine
parent: observation-lifecycle
created: 2026-07-26T12:00:00Z
priority: 1
status: done
depends-on: observations-born-on-branches
---

# Observation Lifecycle State Is Readable Purely via Git — No GitHub API Calls

## Description

The set of `observer/*` branches is the observation state machine. Every
lifecycle question is answerable from local git state after a `git fetch`;
no tooling or prompt consults the GitHub (or any forge) API to determine
observation state.

## Success Criteria

- The documented state mapping is:
  - `observer/<slug>` branch visible locally or on origin → finding is
    announced/pending
  - branch merged to main → resolved: the observation lands on main with
    `status: resolved` and the remedy applied in the same merge
  - branch kept but its PR closed → **parked**: still part of the dedup
    union, never re-announced
  - branch deleted → **forgotten**: the union forgets it; if the drift
    persists, the next scan legitimately re-finds it
- `git fetch` is the **only** remote operation any observer tooling performs
  to read state. No code path in the observer scan, consolidate, or announce
  flow calls a forge API (`gh`, GitHub REST/GraphQL, etc.) to determine
  lifecycle state. PR open/closed status is deliberately invisible and
  irrelevant to the state machine — parked and pending are distinguished by
  human convention, not by tooling.
- Because parked and pending branches are indistinguishable to tooling by
  design, nothing in the system requires distinguishing them: dedup treats
  both identically (suppress re-flagging), and announce idempotency is keyed
  solely off the `announced:` frontmatter marker.
- The state mapping and the fetch-only rule are documented in the parent spec
  and in the observer prompt.

**Note:** "visible" means the branch exists as a local branch or a
remote-tracking ref after `git fetch` — the same visibility rule
`internal/crossbranch` branch discovery already uses.
