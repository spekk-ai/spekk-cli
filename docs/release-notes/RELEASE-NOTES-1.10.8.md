# Spekk CLI 1.10.8 — Dev-Loop Skill for Every Harness

`spekk install` now writes the `spekk-dev-loop` orchestration skill into **every** supported harness, not just Claude Code. The dev loop is the outer coach → coordinate → builders → review pipeline that turns an idea into shipped, verified code on a spekk project — previously only `--target claude-code` installed it, so users of other assistants had to hand-copy it.

## What you get per target

One embedded source, written in whatever form the harness uses for a reusable, agent-invokable workflow:

| Target | Written as | Location |
|--------|-----------|----------|
| `claude-code` | native skill (verbatim) | `~/.claude/skills/spekk-dev-loop/SKILL.md` |
| `opencode` | native skill (verbatim) | `~/.config/opencode/skills/spekk-dev-loop/SKILL.md` |
| `cursor` | `/spekk-dev-loop` command | `~/.cursor/commands/spekk-dev-loop.md` |
| `codex` | `/spekk-dev-loop` prompt | `~/.codex/prompts/spekk-dev-loop.md` (global only) |
| `copilot` | `/spekk-dev-loop` prompt | `.github/prompts/spekk-dev-loop.prompt.md` (`--project` only) |

- **Native-skill harnesses** (claude-code, opencode) share the same `SKILL.md` schema, so they get the skill **byte-for-byte** and the model can invoke it automatically.
- **Command/prompt harnesses** (cursor, codex, copilot) render a whole file as a prompt, so they get the skill body with its YAML frontmatter stripped — a manually-invoked `/spekk-dev-loop` command.
- **Copilot** is project-only (personal prompts are IDE-managed — no fabricated global path); **codex** is global-only. Each simply opts out of the scope it doesn't support.

`--project` writes to the project's equivalent directory instead of the global one, exactly like the agent shims.

## Upgrade

```bash
spekk update
```

Then re-run `spekk install --target <your-harness>` to pick up the dev-loop skill.
