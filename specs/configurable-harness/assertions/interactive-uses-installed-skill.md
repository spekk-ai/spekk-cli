---
id: interactive-uses-installed-skill
parent: configurable-harness
created: 2026-09-06T00:00:00Z
priority: 1
status: in_progress
depends-on: install-covers-hermes-and-gemini
branch: feature/configurable-harness
locked-by: builder-MacBook-Pro-local-96959-1788728588
---

# `spekk coach/builder --harness <h>` self-installs instructions and opens a governed session

`spekk coach --harness <h>` and `spekk builder --harness <h>` are the single
entry point — the user never runs `spekk install` by hand. spekk ensures the
agent instructions are present for the harness, then opens an interactive session
governed by them that waits for input.

Only claude-code has an inline system-prompt (`--system-prompt`). The other
harnesses execute any prompt handed to them, so the instructions must live in
each harness's native auto-loaded mechanism, not in an inline argument.

## Success Criteria

- Running `spekk coach --harness <h>` / `spekk builder --harness <h>` requires no
  prior `spekk install`: the command ensures the instructions are present itself,
  then launches — a single command end to end.
- Instructions are installed via each harness's native mechanism, verified per
  harness: `claude-code` uses inline `--system-prompt` (no install); `opencode`
  and `hermes` use a skill file (hermes loads it with `chat -s`); `gemini` and
  `codex` use an auto-read context file (`GEMINI.md`, `AGENTS.md`). The concrete
  mechanism for each is confirmed against that harness, not assumed.
- The install/ensure step is idempotent: present, up-to-date instructions are not
  rewritten, and it does not clobber unrelated user content in a shared context
  file.
- The full agent prompt is never passed as an inline message the harness
  executes. Any message passed is a short activation, not the prompt body.
- The session waits for user input — it does not auto-run a task and does not exit
  after a single turn.
- spekk stays the launcher and the CLI the agent calls; delivery is via skills or
  context files only — no agent-shim files.
- A test asserts that for a non-claude harness the interactive argv carries no
  full-prompt-body argument and that the ensure-instructions step runs before
  launch.
