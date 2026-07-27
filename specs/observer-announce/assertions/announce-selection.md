---
id: announce-selection
parent: observer-announce
created: 2026-07-26T12:00:00Z
priority: 1
status: done
depends-on: cross-branch-observation-indexing
---

# Announce Selects Up to Three Observations: Unannounced Open, High/Medium Only, Oldest-First Within Severity

## Description

`spekk observer announce` deterministically computes the observations to
announce from the index, after a fetch. Selection rules and the caps live in
Go code, not in a prompt.

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
  observations are the first in this order.
- **Hard caps in code: at most ONE message per invocation, and the message
  carries at most THREE findings.** Findings past the cap stay unannounced
  and wait for the next run. With zero candidates the command does nothing
  announce-related and exits 0 (silence is a valid, successful outcome).
- Selection is deterministic: the same repo/branch state yields the same
  choice (ties on identical `created` broken by a stable key, e.g. slug).
- Running on a schedule (cron) is safe by construction: each run sends at
  most one message with at most three findings, so a backlog drains a few
  findings per run rather than flooding the channel.

**Decision (recorded):** unpushed local observer branches are SKIPPED —
only branches visible on origin are announce-eligible. Pushing the branch
is the scan's job, not announce's; a skipped local-only branch simply waits
for its push. With every candidate skipped the run prints "nothing to
announce" and exits 0.

**Decision (recorded, 2026-07-27):** William changed the cap from one
finding per run to one message per run with at most three findings. Both
posts of the first production run were fine; separate messages for
same-run findings were the problem.

**Tests:** internal/observer/announce_test.go (TestSelectCandidatesRules,
TestAnnounceBatchesEligibleFindingsIntoOneMessage,
TestAnnounceCapsBatchAtThree, TestAnnounceSkipsUnpushedLocalBranches)
