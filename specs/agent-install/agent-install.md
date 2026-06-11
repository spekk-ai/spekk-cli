---
id: agent-install
created: 2026-06-11T00:00:00Z
priority: 1
---

# Agent Install — Use Spekk Prompts from Any Coding Assistant

Users can install the spekk agents (coach, builder, observer) as subagents in their preferred coding assistant (Claude Code, OpenCode, Codex, ...), instead of being limited to `spekk coach` / `spekk builder` launching Claude Code.

## Design

The binary remains the single source of truth for prompts and skills:

- **`spekk prompt <agent>`** prints the layered-resolved agent prompt to stdout. This is the universal escape hatch — it works with any tool that can run a shell command or accept pasted text.
- **`spekk skill list <agent>` / `spekk skill show <agent> <name>`** expose the existing layered `SkillResolver` on the command line, so an agent running in any harness can fetch skill content on demand. Built-in skills ship in the binary; nothing is installed to disk. `.spekk/skills/` (project) and `~/.spekk/skills/` (user) remain optional override layers.
- **`spekk install --target <tool>`** writes a **thin shim** subagent file per agent into the host tool's agent directory. The shim contains only host frontmatter (name/description for auto-delegation) and an instruction to run `spekk prompt <agent>` and adopt the output as operating instructions. Because the real prompt is fetched from the binary at session start, installed agents never go stale and always match the behavior of the `spekk` binary they shell out to.

### Why thin shims (not full prompt copies)

- Copied prompts freeze at install time and silently drift from the binary (`spekk next` behavior, skill names, workflow changes).
- The shim couples prompt version to binary version by construction; updating the binary updates every host.
- Reinstalls are only needed when the shim format itself changes, which is rare by design.

### Non-intrusiveness principles

- Global install is the default — agent definitions are project-agnostic capabilities; a repo opts in by having a `specs/` directory.
- Installed agents must degrade gracefully: if the project has no `specs/` directory or the `spekk` binary is missing, say so briefly and stand down. Never push the workflow on a project that hasn't opted in.
- No new dotfiles are created. Built-in skills are served from the embedded FS.

## Supported targets (v1)

| Target | Global location | Project location (`--project`) |
|--------|----------------|-------------------------------|
| `claude-code` (alias `claude`) | `~/.claude/agents/spekk-<agent>.md` | `.claude/agents/spekk-<agent>.md` |
| `opencode` | `~/.config/opencode/agents/spekk-<agent>.md` | `.opencode/agents/spekk-<agent>.md` |
| `codex` | `~/.codex/prompts/spekk-<agent>.md` | not supported (error) |

Additional targets are added by extending the target registry in `internal/install`.
