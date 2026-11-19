---
id: cli-v1-51-release-notes
parent: release-notes-v1-51
created: 2026-11-19T00:00:00Z
priority: 2
status: done
branch: docs/v1.51.0
---

# Operators Can Read the 1.51.0 Changes

The release notes describe each behavior change after 1.50.0 and give the action needed to use file delivery.

## Success Criteria

- The release note, release notes index, and site navigation include 1.51.0.
- The note describes `spekk list --sort-by`, builder loop `--exit-when-done`, the two new observation types, and the Ctrl+C child wait in `spekk loop builder`.
- The note gives the exact `spekk conversation open --attach` and `spekk artifact upload` forms, the file limits, and the result when the app lacks `agent_conversation_delivery`.
- The note states the Spekk app and Slack bot requirements and states that delivery opens a new Slack message.
- The note gives the operator action and states the protocol version and major from source.
- The CLI reference does not imply that `conversation open` can post into an existing Slack thread.

## Verification

- Compare each claim to source, validate the specs, and build the documentation site.
