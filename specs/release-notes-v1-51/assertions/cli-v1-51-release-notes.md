---
id: cli-v1-51-release-notes
parent: release-notes-v1-51
created: 2026-11-19T00:00:00Z
priority: 2
status: done
branch: docs/v1.51.0
---

# Operators Can Read the 1.51.0 Changes

The release notes describe each behavior change after 1.50.0 and give the action needed to use file delivery and current Slack thread replies.

## Success Criteria

- The release note, release notes index, and site navigation include 1.51.0.
- The note describes `spekk list --sort-by`, builder loop `--exit-when-done`, the two new observation types, and the Ctrl+C child wait in `spekk loop builder`.
- The note states that a 1.50.0 host rejects the new types in `spekk observer scan-check`, skips existing files of those types with a validation warning, and must be upgraded before operators rely on the new types.
- The note gives the exact `spekk conversation open --attach` and `spekk artifact upload` forms, the file limits, and the result when the app lacks `agent_conversation_delivery`.
- The note states the Spekk app and Slack bot requirements, distinguishes a new message from a reply in the current turn, and does not claim that spekk-app #715 is deployed.
- The note gives the exact `spekk conversation reply --body <text> [--attach <file> ...]` form, the capability refusal text, and the result of retrying a partial delivery.
- The note states that `reply` uses only the current turn thread and has no caller-selected channel or thread.
- The note gives the operator action and states the protocol version and major from source.
- The CLI reference explains how `conversation reply` sends to the current turn thread and keeps `conversation open` distinct.

## Verification

- Compare each claim to source, validate the specs, and build the documentation site.
