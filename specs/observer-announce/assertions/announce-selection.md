---
id: announce-selection
parent: observer-announce
created: 2026-07-26T12:00:00Z
priority: 1
status: done
depends-on: cross-branch-observation-indexing
---

# Announce Selects At Most One Observation: Top Unannounced Open, High/Medium Only, Oldest-First Within Severity

## Description

`spekk observer announce` deterministically computes the single observation to
announce from the index, after a fetch. Selection rules and the one-per-run
cap live in Go code, not in a prompt.

## Success Criteria

- `spekk observer announce` is a registered subcommand of the `spekk` CLI
  (`spekk observer announce --help` prints usage).
- The invocation begins with `git fetch` so remote-tracking `observer/*` refs
  are current, then (re)builds/refreshes the index as needed; `git fetch` and
  the final push are the only remote operations — no forge API calls.
- Eligibility: an observation is a candidate iff **all** of:
  - `status: open`
  - frontmatter lacks `announced:` (SQL: `announced IS NULL`)
  - `severity` is `high` or `medium` — `low` NEVER announces, regardless of
    age or queue emptiness
  - it lives on a visible `observer/<slug>` branch
  - evidence gate: it has at least one `affected` path — with none, the
    command refuses to announce it (skips it as ineligible)
- Ordering: candidates sort by severity (`high` before `medium`), then
  oldest-first by `created` within the same severity. The selected
  observation is the first in this order.
- **Hard cap in code: at most ONE announcement per invocation.** With zero
  candidates the command does nothing announce-related and exits 0 (silence
  is a valid, successful outcome).
- Selection is deterministic: the same repo/branch state yields the same
  choice (ties on identical `created` broken by a stable key, e.g. slug).
- Running on a schedule (cron) is safe by construction: each run announces at
  most one finding, so a backlog drains one Slack conversation per run rather
  than flooding the channel.

**Decision (recorded):** unpushed local observer branches are SKIPPED —
only branches visible on origin are announce-eligible. Pushing the branch
is the scan's job, not announce's; a skipped local-only branch simply waits
for its push. With every candidate skipped the run prints "nothing to
announce" and exits 0.

**Tests:** internal/observer/announce_test.go (TestSelectCandidatesRules,
TestAnnounceSelectsHighestSeverityOldestFirst,
TestAnnounceSkipsUnpushedLocalBranches)
