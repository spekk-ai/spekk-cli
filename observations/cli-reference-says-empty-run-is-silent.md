---
slug: cli-reference-says-empty-run-is-silent
type: code_spec_misalignment
severity: medium
status: open
created: 2026-11-07T12:00:00Z
announced: 2026-11-07T20:01:36Z
affected:
  - docs/cli-reference.md
  - specs/observer-agent/assertions/one-observation-per-run.md
  - specs/observer-agent/assertions/digest-as-default-surface.md
---

# CLI reference still says an observer run with an empty digest says nothing

## Issue Description
#347 brought `digest-as-default-surface` and the observer-agent spec in line
with `one-observation-per-run`. A scan prints exactly one line, also when the
digest is empty, and `spekk observer digest` is the wider view, not the run's
report. The user-facing observer docs were not updated. They still describe the
old contract: the run reports from the digest and is silent when it is empty.

## Evidence
- `docs/cli-reference.md:601`: "A run reports a short summary from the
  rendered digest (`spekk observer digest`). There is no committed digest
  file. When the digest is empty, the observer says nothing."
- `specs/observer-agent/assertions/one-observation-per-run.md` (status
  `done`): "A run always prints exactly one line. A run that prints nothing
  cannot be told apart from a run that never happened". Its Verification is
  "Review of `specs/observer-agent/observer.prompt.md` and the observer docs."
- `specs/observer-agent/assertions/digest-as-default-surface.md` (changed in
  #347): the prompt names `spekk observer digest` "as the wider view of open
  observations, and not as the run's report: a scan prints the one line ...
  also when the digest is empty".
- `specs/observer-agent/observer.prompt.md:261` and `:276` already say this.
  The docs are the only surface that disagrees.

## Impact
An operator reading the reference will think a silent run is normal. A run
that never started, or one whose filing failed, will look like an empty
digest. That is exactly the confusion the one-line rule exists to prevent.
The paragraph also presents the digest as the run's report, which the
assertions no longer allow.

## Recommendation
Rewrite the paragraph at `docs/cli-reference.md:601`. A scan prints one line
saying what it did: it filed a finding (named, with severity), it found nothing
in the areas it covered, or it had nothing to scan. `spekk observer digest` is
the separate, wider view of open observations. The remedy flips
`one-observation-per-run` back to `failed` so the builder makes this docs edit.