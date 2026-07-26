---
id: announce-failure-log
parent: observer-announce
created: 2026-07-26T12:00:00Z
priority: 1
status: not_started
depends-on: announce-conversation-open
---

# Announce Failures Append to `.spekk/observer-conversation.log` and Exit Non-Zero Without the Frontmatter Flip

## Description

A headless, cron-driven step must never fail silently — that is the exact
failure mode this redesign exists to kill. Every announce failure leaves two
traces: a non-zero exit code for the scheduler and an appended line in a local
log file for the human who investigates.

## Success Criteria

- Any failure in the announce flow — fetch failure, index failure, spool env
  unset, conversation-request write failure, commit/push failure — causes the
  command to:
  - append a line to `.spekk/observer-conversation.log` containing at least a
    timestamp, the observation slug (when one was selected), and the error;
    the file is created if absent, and appended to (never truncated)
  - exit non-zero
  - leave the observation's frontmatter **without** the `announced:` flip
    (except the delivered-then-push-failed case documented in
    `announce-marks-branch`, which still logs and exits non-zero)
- The log records failures only — successful runs (including the valid
  "nothing to announce" outcome) do not write to it, so a non-empty log
  always means something needs attention.
- `.spekk/observer-conversation.log` is gitignored (it is local operational
  telemetry, not repo state — repo state lives in branches and frontmatter).
- Headless-safety: no code path in `spekk observer announce` prompts for
  input, waits on a TTY, or blocks indefinitely; every failure path
  terminates with the exit-code + log behavior above.

**Note:** three days of silence was the original incident. The invariant this
assertion encodes: an announce that did not happen must be *observable* —
through exit code and log — without anyone remembering to check Slack.
