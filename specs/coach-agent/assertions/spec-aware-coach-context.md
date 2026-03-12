---
id: spec-aware-coach-context
parent: coach-agent
created: 2026-03-01T17:00:00Z
priority: 1
status: not_started
---

# Serve uses the real coach agent prompt and coach reads specs on first interaction

The serve command spawns Claude using the same coach agent prompt that `spekk coach` uses (via `launchAgentWithPrompt`), not a hardcoded duplicate. The extension sends an `init` message on connect with client context. The CLI passes it through like any other message. The coach reads specs from the filesystem itself using its own tools.

## Success Criteria

- `serve/index.js` uses `launchAgentWithPrompt('coach-agent')` from `prompt-resolver.js` to get the coach agent prompt instead of importing from `coach-prompt.js`
- `serve/coach-prompt.js` is deleted — no duplicate prompt
- `message-formatter.js` handles the `init` message type and formats it as readable text for Claude (not dropped, not returned as null)
- The CLI forwards the formatted init message to Claude's stdin like any other message (no special handling — dumb pipe)
- The coach agent prompt (`specs/coach-agent/coach-agent.prompt.md`) instructs Claude to read the `specs/` directory on its first interaction to understand the current spec landscape
- The coach agent prompt instructs Claude to default toward creating or updating specs after understanding a request
- The coach agent prompt instructs Claude to check existing spec groups before proposing new ones
- The coach agent prompt allows one-off questions and lightweight conversations without forcing everything into a spec
