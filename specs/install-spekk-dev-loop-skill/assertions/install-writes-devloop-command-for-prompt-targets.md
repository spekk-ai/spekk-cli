---
id: install-writes-devloop-command-for-prompt-targets
parent: install-spekk-dev-loop-skill
created: 2026-07-15T00:00:00Z
priority: 2
status: not_started
depends-on: install-writes-skill-for-claude-code
---

# `cursor`/`codex`/`copilot` Get a Frontmatter-Stripped `/spekk-dev-loop` Command

The command/prompt harnesses render a whole file as a prompt and (cursor)
forbid YAML frontmatter, so they receive the skill **body only** as a manually
invoked `/spekk-dev-loop` command / prompt file. This assertion owns the shared
`stripFrontmatter` helper and the three command-target descriptors, on top of
the descriptor-driven writer from `install-writes-skill-for-claude-code`.

## Success Criteria

### The `stripFrontmatter` helper (shared, owned here)

- A single `stripFrontmatter([]byte) []byte` helper removes a leading YAML
  frontmatter block: everything from the leading `---` line through the closing
  `---` line and the single blank line that follows it. Applied to the embedded
  source (which starts `---\nname: spekk-dev-loop\ndescription: ...\n---\n\n# Spekk Dev Loop`),
  it returns exactly the body beginning at `# Spekk Dev Loop`.
- It is defensive: content that does not begin with `---\n` is returned
  unchanged. The verbatim targets (claude-code, opencode) never call it.

### Per-target descriptors (all `strip: true`)

- **cursor** writes to both scopes:
  - Global: `<HomeDir>/.cursor/commands/spekk-dev-loop.md`
  - Project (`--project`): `<Cwd>/.cursor/commands/spekk-dev-loop.md`
- **codex** is global-only (it already has no project install — `projectDir: ""`
  for shims, and `--project` errors):
  - Global: `<HomeDir>/.codex/prompts/spekk-dev-loop.md`
  - Project: none (the existing `--project` unsupported error still applies).
- **copilot** is project-only (no standard global filesystem path for a personal
  prompt file — do not invent one):
  - Global: **no dev-loop file** — `globalPath` returns `""`; only the agent
    shims install, and no skill FS is required for a global copilot install.
  - Project (`--project`): `<Cwd>/.github/prompts/spekk-dev-loop.prompt.md`
    (note the `.prompt.md` double extension copilot uses).
- In every written case the file content equals `stripFrontmatter(source)` — the
  skill body with no frontmatter — created `0o755` parent / `0o644` file, path
  appended to the returned slice, idempotent overwrite.

## Tests

Extend `internal/install/install_test.go`:

- Unit-test `stripFrontmatter`: on the fake skill source with frontmatter it
  drops the `---` block and following blank line and returns the body; on input
  without leading `---` it returns the input unchanged. (The fake `SkillFS` used
  here must include a real `---` frontmatter block so the strip is exercised.)
- `cursor` global writes `<home>/.cursor/commands/spekk-dev-loop.md`; `cursor`
  `--project` writes `<cwd>/.cursor/commands/spekk-dev-loop.md`; both contain the
  stripped body and **no** `---` frontmatter line.
- `codex` global writes `<home>/.codex/prompts/spekk-dev-loop.md` (stripped);
  `codex --project` still errors (unchanged, `TestInstall_Errors`).
- `copilot --project` writes `<cwd>/.github/prompts/spekk-dev-loop.prompt.md`
  (stripped); `copilot` global writes **no** dev-loop file — returns only the 3
  shim paths and needs no `SkillFS`.

**Update existing tests broken by the generalization:**

- The `TestInstall_SkillFile` "non-claude-code target produces no skill file"
  subtest (currently asserting `cursor` writes exactly 3 files and no
  `skills/` dir) is now false — `cursor` writes a `/spekk-dev-loop` command.
  Replace it: assert `cursor` writes a stripped command file, and pick a
  genuinely-no-dev-loop case (global `copilot`) for the "3 shims only, no extra
  file" check.
- In `TestInstall_Targets`, the `cursor` (both scopes) and `codex` (global) rows
  now write a dev-loop file and must inject a `SkillFS`; the `copilot` global row
  writes none and stays `nil`. If a `copilot` project row is added, it needs a
  `SkillFS`.
