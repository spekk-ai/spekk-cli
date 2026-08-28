---
id: install-command
parent: agent-install
created: 2026-06-11T00:00:00Z
priority: 1
status: done
---

# `spekk install --target <tool>` Writes Thin Shim Subagents

`spekk install --target <tool>` installs spekk's agents (coach, builder, observer) into a host coding assistant as thin shim files that fetch their real instructions from the binary at session start.

## Success Criteria

- `internal/install` package exports `Install(opts Options) ([]string, error)` returning the list of written file paths. `Options` includes `Target`, `Project bool`, `HomeDir`, `Cwd` (injectable for tests).
- Supported targets: `claude-code` (alias `claude`), `copilot`, `cursor`, `opencode`, `codex`:
  - `claude-code`: global `~/.claude/agents/spekk-<agent>.md`, project `.claude/agents/spekk-<agent>.md`; YAML frontmatter with `name` and `description`.
  - `copilot`: global `~/.copilot/agents/spekk-<agent>.agent.md`, project `.github/agents/spekk-<agent>.agent.md`; YAML frontmatter with `name` and `description`; note the `.agent.md` extension.
  - `cursor`: global `~/.cursor/agents/spekk-<agent>.md`, project `.cursor/agents/spekk-<agent>.md`; YAML frontmatter with `name` and `description`.
  - `opencode`: global `~/.config/opencode/agents/spekk-<agent>.md`, project `.opencode/agents/spekk-<agent>.md`; YAML frontmatter with `description` and `mode: subagent`.
  - `codex`: global `~/.codex/prompts/spekk-<agent>.md`; `--project` returns an error explaining it is not supported for codex.
- Shim body (shared across targets) instructs the agent to:
  - run `spekk prompt <agent>` and adopt the output as its operating instructions for the session, and
  - if the `spekk` command is not found, tell the user to install spekk (linking the repo) and stop.
- Each agent's frontmatter `description` scopes auto-delegation to spec-driven-development requests in projects containing a `specs/` directory, so the agents stay dormant elsewhere.
- Directories are created as needed; existing shim files are overwritten (idempotent re-install).
- `spekk install` with a missing or unknown `--target` prints an error listing valid targets and pointing at `spekk prompt <agent>` as the fallback for unlisted tools, then exits 1.
- Long-tail guidance lives in `spekk install --help` (OTHER TOOLS section) and the README ("Use Spekk from Your Coding Assistant"): any tool can consume `spekk prompt <agent>` directly via its custom-agent/rules mechanism or an `AGENTS.md` line.
- `spekk install --help` shows usage; `spekk help` lists the `install` command.
- On success the command prints each written path.

**Tests:** `internal/install/install_test.go` — written paths per target/scope, shim content (frontmatter + body), codex `--project` error, unknown target error, idempotent overwrite.
