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
- A run ends when it files its observation, **or** when it has covered the
  areas it set out to cover — whichever comes first. Finding nothing is a
  valid outcome and ends the run.
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
