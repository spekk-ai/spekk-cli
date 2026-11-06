---
slug: digest-assertion-mandates-silent-run
type: outdated_specs
severity: medium
status: open
created: 2026-11-06T20:02:05Z
announced: 2026-11-06T20:03:38Z
affected:
  - specs/observer-agent/assertions/digest-as-default-surface.md
  - specs/observer-agent/assertions/one-observation-per-run.md
  - specs/observer-agent/observer-agent.md
  - specs/observer-agent/observer.prompt.md
---

# Digest assertion still says a scan stays silent when the digest is empty

## Issue Description

Two `done` assertions under `specs/observer-agent` give opposite rules for how a scan reports.
`digest-as-default-surface` says a scan reports through `spekk observer digest` and stays silent when the digest is empty.
`one-observation-per-run` is the later assertion (#176). It says a run always prints exactly one line, because a silent run cannot be told apart from a run that never happened.
The prompt follows the later rule.
The older assertion and the "Silent suppression" principle in the parent spec were never updated.

## Evidence

- `specs/observer-agent/assertions/digest-as-default-surface.md`, Success Criteria:
  - "The prompt directs reporting through the rendered view: run `spekk observer digest` and print at most a one-line summary of its output"
  - "Silent suppression: when the rendered digest is empty, the agent says nothing to the user for that cycle"
- `specs/observer-agent/assertions/one-observation-per-run.md`, line 39: "A run always prints exactly one line. A run that prints nothing cannot be told apart from a run that never happened, and silence would hide a filing that failed."
- `specs/observer-agent/observer.prompt.md`, line 261: "A scan prints exactly one line, always." Line 276: "`spekk observer digest` ... is not this run's report, and its being empty is not a reason to stay silent."
- `specs/observer-agent/observer-agent.md`, line 49: "**Silent suppression.** A scan with nothing worth saying produces nothing user-facing."

## Impact

Both assertions are `done`, and they cannot both be satisfied.
If a builder or reviewer checks the prompt against `digest-as-default-surface`, it fails, and a fix to that assertion would bring back the silent run that `one-observation-per-run` removed.
A reader of the parent spec gets the old principle and no sign that it was replaced.

## Recommendation

Bring the older assertion and the parent principle in line with `one-observation-per-run`.
A scan prints one line about what it did, always, and the digest is an optional wider view, not the report.
The "speak rarely" goal survives, because announce carries only `high` and `medium` findings to chat.
The remedy commit on this branch makes that spec edit.