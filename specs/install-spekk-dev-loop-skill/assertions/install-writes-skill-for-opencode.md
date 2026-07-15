---
id: install-writes-skill-for-opencode
parent: install-spekk-dev-loop-skill
created: 2026-07-15T00:00:00Z
priority: 2
status: done
depends-on: install-writes-skill-for-claude-code
---

# `spekk install --target opencode` Writes the `spekk-dev-loop` Skill Verbatim

OpenCode has a native skills directory using the same `name` + `description`
`SKILL.md` frontmatter schema as Claude Code, so it is the second
native-verbatim target. It reuses the descriptor-driven writer from
`install-writes-skill-for-claude-code` and only adds its own destination.

## Success Criteria

- `opencode`'s target descriptor sets `strip: false` and writes the embedded
  skill **verbatim** (byte-for-byte, frontmatter + body) to opencode's native
  skills location — note these are distinct from the `.config/opencode` /
  `.opencode` **agents** dirs the shims already use:
  - Global (no `--project`): `<HomeDir>/.config/opencode/skills/spekk-dev-loop/SKILL.md`
  - Project (`--project`): `<Cwd>/.opencode/skills/spekk-dev-loop/SKILL.md`
- The parent directory is created `0o755`; the file is written `0o644`.
- The written path is appended to the slice `Install` returns (printed like the
  shims).
- The written bytes equal the injected skill bytes exactly — no transform.
- Re-install overwrites in place silently (idempotent).

## Tests

Extend `internal/install/install_test.go`, injecting `HomeDir`/`Cwd` and a fake
`SkillFS`:

- `opencode` global writes `<home>/.config/opencode/skills/spekk-dev-loop/SKILL.md`
  whose bytes equal the injected skill bytes; the path is in the returned slice.
- `opencode` with `--project` writes `<cwd>/.opencode/skills/spekk-dev-loop/SKILL.md`
  (not under home).
- The existing `TestInstall_Targets` `opencode` rows (`false` and `true`) must
  now inject a `SkillFS` — both scopes write a skill — so the fail-loudly path
  doesn't error them out.
