---
id: interactive-uses-installed-skill
parent: configurable-harness
created: 2026-09-06T00:00:00Z
priority: 1
status: done
depends-on: harness-profile-abstraction
branch: feature/configurable-harness
---

# Interactive coach/builder is governed by the installed skill, not an inline prompt

**Tests:** internal/agent/harness_test.go, internal/agent/launcher_test.go, internal/install/ensure_role_skill_test.go

Harnesses other than claude-code have no inline system-prompt mechanism: any
prompt handed to them is a message they execute. Passing the full agent prompt
that way is what breaks them — opencode auto-runs it as a build task, hermes
answers once and exits. Instead the instructions live in the spekk skill, which
spekk ensures is installed, and the harness opens an interactive session
governed by that skill.

## Success Criteria

- Launching interactive `spekk coach` / `spekk builder` on a non-claude harness
  first ensures the relevant spekk skill (`spekk-coach` / `spekk-builder`) is
  installed for that harness — the same result as `spekk install --target <h>` —
  installing it if absent. The check is idempotent: an up-to-date skill is not
  rewritten.
- The full agent prompt is never passed as an inline message argument. Any
  message passed is a short activation that loads the skill, not the prompt body.
- The harness then opens an interactive session that waits for user input; it
  does not auto-execute a task and does not exit after a single turn.
- `claude-code` is unaffected: it keeps its inline `--system-prompt` path and
  needs no skill install to run interactively.
- spekk stays the launcher and the CLI the agent calls; only prompt delivery
  changes. Delivery is via the installed skill — no agent-shim files are written
  or required for this path.
- A test asserts that for a non-claude harness the resolved interactive argv
  carries no full-prompt-body argument, and that the skill-install step runs
  before launch.
