# Spekk CLI 1.10.5 — Dev-Loop Skill + Coach Declarative Rewrite

This release bundles a Claude Code skill with `spekk install` and tightens the coach's declarative framing.

## `spekk-dev-loop` skill for Claude Code

`spekk install --target claude-code` now writes a `spekk-dev-loop` skill alongside the three agent shims:

```
$ spekk install --target claude-code
installed: ~/.claude/agents/spekk-coach.md
installed: ~/.claude/agents/spekk-builder.md
installed: ~/.claude/agents/spekk-observer.md
installed: ~/.claude/skills/spekk-dev-loop/SKILL.md   # new
```

The skill is embedded in the binary and describes the outer dev loop — a coach → coordinate → builders → review pipeline for building features on a spekk-enabled project. Use `--project` to write it to `.claude/skills/spekk-dev-loop/SKILL.md` instead. Only the `claude-code` target writes the skill.

## Coach: declarative framing rewrite

The coach prompt's declarative-framing section (COACH-06) was rewritten:

- **Write-first rule** — when you say "write a spec for X", the coach writes the assertion first, with no preface and no clarifying questions before the output. Follow-up questions come after, never before.
- **Tighter declarative guidance** — a verbose imperative→declarative explanation is replaced with a concise instruction and a three-example table (assertions declare state, not actions).
- **`Done when:` block** — the coach now closes every proposal with a short, builder-verifiable checklist of what the system *is* after the change ships, not what the change *does*.

## Repository hygiene

Cross-compiled binaries that had been committed to the repository root by accident are removed and gitignored; two obsolete agent-loop shell scripts (superseded by `spekk builder` / `spekk coach`, which loop by default) are deleted; and the Node→Go migration guide moved into the docs site.

The release workflow now builds and uploads binaries automatically on any `v*` tag (previously only `exp-*` experimental tags auto-built, so stable releases required a manual workflow run — which was occasionally missed, shipping releases with no downloadable assets).

## Upgrade

```bash
spekk update
```
