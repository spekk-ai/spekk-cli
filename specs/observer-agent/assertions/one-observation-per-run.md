---
id: one-observation-per-run
parent: observer-agent
created: 2026-08-08T00:00:00Z
priority: 1
status: done
---

# A Scan Run Files One Observation and Ends

A single `spekk observer` run files at most one lifecycle observation, and
ends by itself. It never runs until something else stops it.

The rate at which observations arrive is set by the schedule that invokes the
observer — `install-cron`, or whatever else dispatches it — and never by how
long a run continues. The two are independent: shortening the interval does
not make a run more thorough, and a run does not decide how often it happens.

## Why

The observer once described an infinite monitoring loop, so the schedule was
the only thing bounding a session. One production run scanned for nearly two
hours, fanned out across many subagents, and filed nine observations at once.
The findings were good; the cost and the review load were not, and a later run
would have found the same drift.

One observation is the atomic unit: one branch, one PR, one thing to decide
about. Nothing is lost by stopping, because drift found today is still there
tomorrow.

## Success Criteria

- The observer prompt states the cap as its first workflow rule, before any
  step that could file something.
- A run ends when it files its observation — branch, commits, push, and PR
  complete — **or** when it has covered the areas it set out to cover,
  whichever comes first. Finding nothing is a valid outcome and ends the run.
  There is no third ending, and nothing after either one returns to scanning.
- A run always prints exactly one line. A run that prints nothing cannot be
  told apart from a run that never happened, and silence would hide a filing
  that failed.
- The areas come from about a week of commits on `origin/main`. The window is
  longer than the interval on purpose: a run files one observation, so a
  second finding in the same commits stays in scope for later runs. The
  choice is bounded at both ends — never the whole repository, never so
  little that the run cannot fail — and it skips areas that carry an open
  observation or a suppression, because those cannot produce a filing.
- A repository with no recent commits ends the run without scanning. An idle
  repository costs nothing to observe.
- The prompt states plainly that the observer watches change and does not
  audit the repository. No text may claim or imply that a later run will
  reach drift in code that nobody has touched. That is the coverage record's
  job, and it does not exist yet.
- A scan never curates. Steps that dismiss a finding belong to the
  `consolidate` run, which is invoked and scheduled separately, so a scan
  cannot bury a finding that no person has judged.
- A run names, in its report, the areas it planned to cover but did not, so a
  short run is not mistaken for a clean audit.
- Nothing in the prompt, the CLI, or the docs offers a per-run cadence, and
  no document justifies the schedule's default by the size of the cap.
- `spekk observer announce` keeps its own cap of three findings per message.
  That is a delivery cap, not a scan cap: it drains a backlog, and a run that
  files one does not make it dead.

## Verification

Review of `specs/observer-agent/observer.prompt.md` and the observer docs.
There is no automated check: the cap is prompt-level, and a deterministic
budget belongs to the run-cost work, which is separate.
