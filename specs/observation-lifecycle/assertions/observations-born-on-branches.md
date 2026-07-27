---
id: observations-born-on-branches
parent: observation-lifecycle
created: 2026-07-26T12:00:00Z
priority: 1
status: done
depends-on: observation-file-format
---

# Observations Are Born on `observer/<slug>` Branches, Never Committed to Main by the Observer

## Description

The observer records each finding on a dedicated branch named
`observer/<slug>`, containing the observation file and a proposed remedy as
separate commits. Main only ever receives observations through a human merging
that branch.

## Success Criteria

- The observer workflow (prompt and any Go tooling that assists it) creates
  one branch per finding, named `observer/<slug>` where `<slug>` equals the
  observation's frontmatter `slug`.
- The branch carries the finding as **two separate commits**, in order:
  1. the observation file under `observations/` with `status: open`
  2. the proposed remedy
- The remedy commit's shape depends on `type`:
  - `outdated_specs` (spec-side drift): the actual spec edit under `specs/`
  - `code_spec_misalignment` (code-side drift): **only** the affected
    assertion's frontmatter status flip `done` → `failed` — no code changes
- No observer code path or prompt instruction commits an observation file
  directly to main. On main, observations appear only via merges of observer
  branches.
- The commit separation is stated as a requirement in the observer prompt with
  its rationale: a reviewer can take the observation without the remedy (or
  vice versa) by cherry-picking one commit.

**Note:** the remedy for code-side drift is deliberately minimal — flipping
the assertion to `failed` re-queues it for the builder; the observer never
writes implementation code.
