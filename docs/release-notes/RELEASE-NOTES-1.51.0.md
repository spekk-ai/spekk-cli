# Spekk CLI 1.51.0 - Agent Delivery and Builder Loop Control

This minor release adds file delivery and current Slack thread replies, a sort flag, and a way to stop a builder loop when its work is complete. It also adds two observation types and changes how `spekk loop builder` handles Ctrl+C.

## Agent file delivery

`spekk conversation open --title <text> --body <text> --attach <file>` opens a new Slack message with its text and files. Repeat `--attach` for up to three files. The CLI accepts a regular file of at most 10 MiB for each attachment and at most 20 MiB for all attachments together. In a current sandbox session with direct delivery, the command prints the Slack message link and each delivered file ID. Slack workspace file type rules still apply. `conversation open` starts a new message; use `conversation reply` to send to the current turn thread.

`spekk conversation reply --body <text> [--attach <file> ...]` sends text and up to three files to the Slack thread of the current sandbox turn. `--body` is required. The app selects the thread from the current turn; the caller cannot choose a channel or thread. The same 10 MiB per file and 20 MiB total limits apply. The command prints the Slack message link and delivered file IDs.

`reply` needs an app that advertises `agent_current_thread_delivery` in the welcome frame. Without that capability, it sends no request and fails with `Error: the app does not support current Slack thread delivery; update the app before using reply`. If the text posts but a file fails, the error includes the posted message link when the app provides it. A second `reply` call posts the text again; inspect the first link before retrying.

In 1.51.0, `spekk artifact upload <file> [--title <text>] [--kind <markdown|image|video>]` uploads one file with the sandbox session's agent token and prints its app link. The CLI infers the kind from known extensions, or you can set `--kind`. It accepts a regular file of at most 1 MiB for markdown, 10 MiB for images, or 90 MiB for video. The artifact starts with organization access. If the app returns 404, the command reports that artifacts are off.

The app must advertise `agent_conversation_delivery` to use direct conversation delivery. Without that capability, a plain `spekk conversation open` request still uses the request spool, but `--attach` fails with `the app does not support conversation file delivery; update the app before using --attach`. An older sandbox session without `SPEKK_AGENT_SESSION_ID` can also queue a plain request, but cannot attach files or return a delivery result. A partial delivery error reports that the text posted and gives its link when the app provides one.

## Other changes

- `spekk list --sort-by <column>` sorts by `id`, `priority`, `status`, or `title` in ascending order. It works with the table, JSON, TSV, and CSV formats and with `--status` and `--priority` filters. It does not apply to `--cross-branch`. The default order stays priority, then ID.
- `spekk builder --exit-when-done` and `spekk loop builder --exit-when-done` exit with status 0 and report the successful build count when no open assertion remains. Without the flag, the loops wait for more work. Open assertions that are not eligible still make the loops wait.
- Observations now accept `underspecified_spec` when a spec permits more than one implementation and `missing_capability` when the specs do not give users a needed capability.
- A host on 1.50.0 rejects these types in `spekk observer scan-check` and skips existing files of these types with a validation warning. Upgrade every host that runs observers to 1.51.0 before you rely on the new types.
- On Ctrl+C during an active `spekk loop builder` build, the loop sends the interrupt to its child and waits for the child to finish. A later Ctrl+C when no child is active exits the loop.

## What you must do

Update the CLI to 1.51.0 to use these commands and flags. Direct conversation file delivery needs spekk-app #714. Current thread replies need spekk-app #715 with `agent_current_thread_delivery`; confirm that deployment before using `reply`. For Slack files, the bot token needs `files:write`; add that scope and reinstall the Slack app if the bot does not have it. No configuration change is needed for plain conversation requests.

## Compatibility

The WebSocket protocol version stays `1.6`, and its major stays `1`.
