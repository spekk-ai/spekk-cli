---
id: install-writes-skill-for-claude-code
parent: install-spekk-dev-loop-skill
created: 2026-07-15T00:00:00Z
priority: 1
status: not_started
depends-on: embed-skill-content
---

# `spekk install --target claude-code` Writes the `spekk-dev-loop` Skill

When installing into Claude Code, `Install` reads the embedded `spekk-dev-loop`
skill and writes it to disk alongside the agent shims. No other target writes a
skill. This assertion owns both the FS-injection wiring into `internal/install`
and the write behavior.

## Success Criteria

- **FS injection.** `internal/install` reads the embedded skill through an
  injectable filesystem, mirroring the existing prompt/skill wiring in
  `internal/cli`: a package-level `install.DefaultSkillFS fs.FS`, set in
  `main.go` to `spekk.EmbeddedFS` right beside the existing
  `cli.DefaultEmbeddedFS` / `cli.DefaultEmbeddedSkillFS` assignments, plus an
  `Options` override field (e.g. `SkillFS fs.FS`) that falls back to
  `DefaultSkillFS` when unset — so tests supply a fake FS without the real
  embed. The embedded skill path is referenced through a single named constant,
  not a scattered string literal. (Use a dedicated `install.DefaultSkillFS`,
  not `cli.DefaultEmbeddedSkillFS` — `internal/install` must not depend on
  `internal/cli`.)
- **Fail loudly if unavailable.** If the resolved target is `claude-code` but no
  skill FS is available (both `Options.SkillFS` and `DefaultSkillFS` are nil, or
  the constant path is unreadable), `Install` returns an error rather than
  silently skipping — the skill is not optional for `claude-code`.
- **Write paths.** For the `claude-code` target (and its `claude` alias),
  `Install` writes the skill bytes verbatim to (note: a `skills/` sibling of the
  `agents/` dir, not under it):
  - Global (no `--project`): `<HomeDir>/.claude/skills/spekk-dev-loop/SKILL.md`
  - Project (`--project`): `<Cwd>/.claude/skills/spekk-dev-loop/SKILL.md`
- The skill's parent directory is created as needed (`0o755`); the file is
  written `0o644` — same permissions as the agent shims.
- The written `SKILL.md` path is appended to the slice `Install` returns, so
  `runInstallTargets` prints it with the same `installed: <path>` line as the
  shims (no change needed in `main.go` beyond wiring `install.DefaultSkillFS`).
- The written file bytes equal the bytes read from the skill FS exactly
  (verbatim copy — no transformation, trimming, or re-rendering).
- An existing `SKILL.md` is overwritten silently (idempotent re-install), with
  no prompt or warning — matching agent-shim behavior.
- Non-`claude-code` targets (`copilot`, `cursor`, `opencode`, `codex`) write
  NO skill file, create no `skills/` directory, and their returned path list is
  unchanged from before this feature.

## Tests

Extend `internal/install/install_test.go`, injecting `HomeDir`, `Cwd`, and a
fake `SkillFS` (a known one-file FS at the constant path) so nothing touches the
real home directory or the real embed:

- `claude-code` global install writes
  `<home>/.claude/skills/spekk-dev-loop/SKILL.md` whose bytes equal the injected
  skill bytes, and the path is in the returned slice.
- `claude-code` with `--project` writes
  `<cwd>/.claude/skills/spekk-dev-loop/SKILL.md` (not under home).
- `claude` alias behaves identically to `claude-code`.
- A non-`claude-code` target (e.g. `cursor`) produces no `SKILL.md` and no
  `skills/` directory, and returns the same number of paths as before.
- Re-running the install overwrites an existing `SKILL.md` without error
  (idempotent).

**Update existing tests that assume claude-code writes only 3 files:**

- `TestInstall_ShimContent` asserts `len(written) == 3` and calls
  `Install` for `claude-code` without a `SkillFS`. It must inject a fake
  `SkillFS` and expect 4 returned paths (3 shims + 1 skill) — or scope its
  count check to the 3 shim files.
- The `claude` / `claude-code` rows of `TestInstall_Targets` also call
  `Install` for a claude target; they must inject a `SkillFS` so the new
  fail-loudly path doesn't error them out.
