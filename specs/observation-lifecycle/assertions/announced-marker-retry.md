---
id: announced-marker-retry
parent: observation-lifecycle
created: 2026-07-26T12:00:00Z
priority: 1
status: not_started
depends-on: observations-born-on-branches
---

# The Slack-Sent Marker Is a Frontmatter Commit on the Observer Branch, and Its Absence Drives Idempotent Retry

## Description

Whether a finding has been announced to Slack is recorded declaratively: a
commit on the `observer/<slug>` branch that sets `announced:` in the
observation's frontmatter. There is no ledger file. Recovery from a failed or
interrupted announce is therefore a pure function of git state.

## Success Criteria

- After a Slack conversation opens successfully for an observation, exactly
  one additional commit on its `observer/<slug>` branch sets
  `announced: <ISO 8601 timestamp>` in the observation frontmatter (and is
  pushed).
- The marker is written **only after** the conversation open succeeds — a
  failed send leaves the frontmatter untouched.
- Retry rule: an `observer/<slug>` branch whose observation frontmatter lacks
  `announced:` is eligible for announcement. Re-running the announce flow
  after a crash mid-send either re-sends (if the flip was never committed) or
  skips (if it was) — both outcomes are correct, and the worst case is one
  duplicate Slack pointer, never a silent drop.
- No file plays the role of an announce ledger: `observations/announced.log`
  (or any equivalent) does not exist in the workflow, and the observer prompt
  contains no instruction to maintain one.

**Note:** this inverts the failure mode of the interim ledger design — there,
losing an untracked ledger file caused re-announce storms or silence; here the
marker travels with the observation on a pushed branch, so it survives clone
hygiene, fresh checkouts, and sandbox rebuilds.
