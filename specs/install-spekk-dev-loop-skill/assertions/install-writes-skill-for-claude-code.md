---
id: install-writes-skill-for-claude-code
parent: install-spekk-dev-loop-skill
created: 2026-07-15T00:00:00Z
priority: 1
status: not_started
depends-on: embed-skill-content
---

# Descriptor-Driven Skill Write, With `claude-code` as the Verbatim Reference

`Install` writes the embedded `spekk-dev-loop` skill through a single
per-target descriptor rather than a `claude-code`-only special case. This
assertion owns the generalized write mechanism (descriptor field, write loop,
FS injection, fail-loudly) and pins `claude-code`'s native-verbatim behavior as
the reference target. The other harnesses (`opencode`;
`cursor`/`codex`/`copilot`) are covered by sibling assertions that depend on
this one and only add their descriptor data + transform.

The `if name == "claude-code"` block in `Install()` (writing `SKILL.md`) is
replaced by descriptor-driven logic; the previous "no other target writes a
skill" guarantee is intentionally gone.

## Success Criteria

### FS injection (unchanged from the original claude-code-only design)

- `internal/install` reads the embedded skill through an injectable filesystem:
  a package-level `install.DefaultSkillFS fs.FS`, set in `main.go` to
  `spekk.EmbeddedFS` beside the existing `cli.DefaultEmbeddedFS` /
  `cli.DefaultEmbeddedSkillFS` assignments, plus an `Options.SkillFS fs.FS`
  override that falls back to `DefaultSkillFS` when unset — so tests supply a
  fake FS without the real embed. The embedded skill path is referenced through
  the single named constant `skillEmbedPath`, not a scattered string literal.
  (Use `install.DefaultSkillFS`, not `cli.DefaultEmbeddedSkillFS` —
  `internal/install` must not depend on `internal/cli`.)

### The generalized skill destination + write loop

- The `target` struct gains a skill destination: `globalPath func(home string) string`,
  `projectPath func(cwd string) string`, and `strip bool`. Either path func may
  be `nil`/return `""` to opt that scope out of writing a dev-loop file.
- After writing the agent shims, `Install()` resolves the descriptor's path for
  the active scope — `globalPath(home)` for a default install, `projectPath(cwd)`
  for `--project`. If the resolved path is `""`, no dev-loop file is written and
  the skill FS is not read (no error, no directory created).
- When a path is resolved, `Install()` reads the embedded bytes via
  `skillEmbedPath` through the injected FS, applies the transform (verbatim when
  `strip` is false; frontmatter-stripped when true — the strip helper itself is
  owned by `install-writes-devloop-command-for-prompt-targets`), creates the
  parent directory `0o755`, writes the file `0o644`, and appends the written
  path to the slice `Install` returns — so `runInstallTargets` prints it with the
  same `installed: <path>` line as the shims (no change needed in `main.go`
  beyond wiring `install.DefaultSkillFS`).
- **Fail loudly.** If a dev-loop file is due for the active target+scope (path is
  non-empty) but no skill FS is available (both `Options.SkillFS` and
  `DefaultSkillFS` are nil, or `skillEmbedPath` is unreadable), `Install`
  returns an error rather than silently skipping. Scopes that opt out (path
  `""`) never require an FS and never error.
- **Only `claude-code` is populated by this assertion.** At the end of this
  assertion, `claude-code` is the sole target with a non-empty skill
  destination. Every other target (`opencode`, `cursor`, `codex`, `copilot`)
  leaves both `globalPath` and `projectPath` `nil`/returning `""` — opted out —
  so after this assertion alone they write only their 3 shims, never read the
  skill FS, and never hit the fail-loudly path. The sibling assertions fill in
  each of those targets' descriptors; this assertion must not.

### `claude-code` reference behavior (native, verbatim)

- `claude-code`'s descriptor sets `strip: false` and writes to (a `skills/`
  sibling of the `agents/` dir, not under it):
  - Global (no `--project`): `<HomeDir>/.claude/skills/spekk-dev-loop/SKILL.md`
  - Project (`--project`): `<Cwd>/.claude/skills/spekk-dev-loop/SKILL.md`
- The `claude` alias resolves to `claude-code` and behaves identically.
- The written `SKILL.md` bytes equal the bytes read from the skill FS exactly —
  no transformation, trimming, or re-rendering.
- An existing `SKILL.md` is overwritten silently (idempotent re-install), with
  no prompt or warning — matching agent-shim behavior.

## Tests

Extend `internal/install/install_test.go`, injecting `HomeDir`, `Cwd`, and a
fake `SkillFS` (a known one-file FS at `skillEmbedPath`) so nothing touches the
real home directory or the real embed:

- `claude-code` global install writes
  `<home>/.claude/skills/spekk-dev-loop/SKILL.md` whose bytes equal the injected
  skill bytes, and the path is in the returned slice.
- `claude-code` with `--project` writes
  `<cwd>/.claude/skills/spekk-dev-loop/SKILL.md` (not under home).
- `claude` alias behaves identically to `claude-code`.
- Re-running the install overwrites an existing `SKILL.md` without error.
- With a target+scope that writes a skill and both `Options.SkillFS` and
  `DefaultSkillFS` nil, `Install` returns an error (fail-loudly).

**Update existing tests broken by the generalization:**

- `TestInstall_ShimContent` (claude-code, `SkillFS` injected) still expects 4
  returned paths (3 shims + 1 skill) — or scopes its count check to the 3 shim
  files.
- `TestInstall_Targets` needs no row changes in this assertion. Because only
  `claude-code` is populated here (every other target opted out), only the
  `claude`/`claude-code` rows write a dev-loop file, and those rows already
  inject a `SkillFS`. Every non-`claude-code` row keeps `skillFS: nil` and stays
  green — those targets are opted out, so they never read the FS and never hit
  the fail-loudly path. **Do not add `SkillFS` to non-`claude-code` rows here.**
  The sibling assertions each own updating only their own target's rows when they
  populate that descriptor: `install-writes-skill-for-opencode` updates the
  `opencode` rows; `install-writes-devloop-command-for-prompt-targets` updates
  the `cursor` and `codex` rows (and, if added, a `copilot --project` row).
- Likewise, the `TestInstall_SkillFile` "non-claude-code target produces no skill
  file" subtest (asserting `cursor` writes exactly 3 files) is still true after
  this assertion — `cursor` is opted out here — so this assertion does not touch
  it. `install-writes-devloop-command-for-prompt-targets` owns replacing it once
  `cursor` starts writing a command.
