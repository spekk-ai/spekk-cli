---
id: sandbox-result-usage
created: 2026-10-24T02:14:00Z
priority: 1
branch: feature/result-usage
---

# A sandbox result reports the model use of its turn

## Problem

The control host cannot show the model use of a sandbox turn because the client ends the turn without a usage value. Claude Code provides token counts and a cost total in its result event. On Claude Code 2.1.308, a resumed process reports a running session cost total, so the client must measure the difference from the preceding process.

## Decision

The client adds one optional `usage` object to a successful turn's result frame. It keeps the last valid cost total for each Claude session while the client runs. It omits usage if a resumed process has no known baseline. It does not keep a guessed cost across a client restart.

## Scope

The client does not add usage to error frames, change the existing result fields, or change the protocol major. The control host accepts an absent usage field.
