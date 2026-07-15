---
id: install-spekk-dev-loop-skill
created: 2026-07-15T00:00:00Z
priority: 2
---

# Install the `spekk-dev-loop` Claude Code Skill

`spekk install --target claude-code` should also install the `spekk-dev-loop`
orchestration skill into the user's Claude Code setup, so anyone who installs
spekk gets the outer coach → coordinate → builders → review dev loop without
hand-copying `SKILL.md` files.

## Problem

The `spekk-dev-loop` skill is the outer loop that drives the coach and builder
agents through a spec-driven feature end to end. Today it only exists as a
hand-placed file at `~/.claude/skills/spekk-dev-loop/SKILL.md`. There is no way
to distribute it: a fresh spekk install gives you the agent shims but not the
skill that orchestrates them. This closes that gap by piggybacking on the
existing `spekk install --target claude-code` flow.

## Design

Follow the established `internal/install` pattern (thin agent shims), with one
addition scoped strictly to the `claude-code` target:

- **Embed the skill in the binary.** A verbatim copy of the skill lives at
  `specs/install-spekk-dev-loop-skill/spekk-dev-loop-skill.md` and is added to
  the `//go:embed` directive in `embedded.go`, mirroring how the coach/observer
  skills are embedded (`specs/<feature>/<name>-skill.md`). The embedded FS is
  injected into `internal/install` the same way it is injected into `internal/cli`
  (`cli.DefaultEmbeddedFS = spekk.EmbeddedFS` in `main.go`), with a per-call
  override field for tests — no new dotfiles, no network fetch.

- **Write it during `spekk install --target claude-code`.** After writing the
  agent shims, when the canonical target is `claude-code`, also write the
  embedded skill content verbatim to:
  - Global (default): `~/.claude/skills/spekk-dev-loop/SKILL.md`
  - Project (`--project`): `.claude/skills/spekk-dev-loop/SKILL.md`

  The written path is included in the returned list, so the command prints it
  like every other installed file.

### Decisions (decided, not asked)

- **Path.** Claude Code discovers skills under `.claude/skills/<name>/SKILL.md`
  (global under `~/`, project under the repo root), so the file is always
  `spekk-dev-loop/SKILL.md` in the `skills` sibling of the `agents` directory
  the shims already target. The name and path are fixed — not configurable.
- **claude-code only.** `skills` are a Claude Code concept. Other targets
  (`copilot`, `cursor`, `opencode`, `codex`) get no skill file. The alias
  `claude` resolves to `claude-code` and behaves identically.
- **Overwrite silently.** Same idempotent behavior as the agent shims — a
  re-install overwrites `SKILL.md` in place with no prompt or warning. The
  skill tracks the binary; a stale copy is never desirable.
- **Embedded source is verbatim.** The written `SKILL.md` (frontmatter + body)
  is a byte-for-byte copy of the embedded source, so the installed skill is
  exactly what ships in the binary.

### Temptations rejected

- No general "skill registry" or arbitrary-skill installer — this ships exactly
  one skill through one existing command.
- No new top-level command; this rides `spekk install --target claude-code`.
- No support for non-claude-code targets, and no configurable skill name/path.

## Assertions

See `assertions/` for what must be true.
