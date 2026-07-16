---
id: install-spekk-dev-loop-skill
created: 2026-07-15T00:00:00Z
priority: 2
---

# Install the `spekk-dev-loop` Skill Into Every Supported Harness

`spekk install --target <harness>` should install the `spekk-dev-loop`
orchestration workflow into whatever native form that harness uses for a
reusable, agent-invokable prompt — a skill where the harness has skills, a
slash command / prompt file where it doesn't — so anyone who installs spekk
gets the outer coach → coordinate → builders → review dev loop without
hand-copying files.

## Problem

The `spekk-dev-loop` skill is the outer loop that drives the coach and builder
agents through a spec-driven feature end to end. It ships embedded in the binary
(`embed-skill-content`, done) and `spekk install` already writes it to disk —
but **only for the `claude-code` target**. Every other harness we support
(`opencode`, `cursor`, `codex`, `copilot`) gets the agent shims and no dev loop.

We want the dev loop installable into all of them. This reverses the original
"claude-code only — skills are a Claude Code concept" decision: `opencode` also
has a native skills directory using the same `SKILL.md` schema, and the
command/prompt harnesses can host the same workflow as a manually-invoked slash
command. The one embedded source should reach every harness through the existing
`spekk install` flow.

## Design

Extend the established `internal/install` pattern. Each target already carries a
descriptor (`globalDir`, `projectDir`, `fileExt`, `frontmatter`) that says where
its agent shims go. Add a **skill destination** to that descriptor and replace
the `claude-code`-only block in `Install()` with a single descriptor-driven
writer that runs for every target.

### The skill destination descriptor

Add one field to the `target` struct — a small value describing where this
harness's dev-loop file goes and how the embedded bytes are transformed:

- `globalPath func(home string) string` — absolute destination for a global
  (default) install, or `""` to write no dev-loop file globally.
- `projectPath func(cwd string) string` — absolute destination for a
  `--project` install, or `""` to write none for project installs.
- `strip bool` — when true, strip the YAML frontmatter from the embedded skill
  before writing (command/prompt harnesses); when false, write the embedded
  bytes verbatim (native-skill harnesses).

A per-scope `""` opt-out (rather than a per-target one) is what lets `codex` be
global-only and `copilot` be project-only without a second mechanism. No
interface hierarchy, no class enum — a path pair plus a bool is enough to
express both the native-skill and the command-file harnesses.

### Two classes of target, one embedded source

| Target | Global (default) | Project (`--project`) | `strip` |
|---|---|---|---|
| claude-code | `<home>/.claude/skills/spekk-dev-loop/SKILL.md` | `<cwd>/.claude/skills/spekk-dev-loop/SKILL.md` | false |
| opencode | `<home>/.config/opencode/skills/spekk-dev-loop/SKILL.md` | `<cwd>/.opencode/skills/spekk-dev-loop/SKILL.md` | false |
| cursor | `<home>/.cursor/commands/spekk-dev-loop.md` | `<cwd>/.cursor/commands/spekk-dev-loop.md` | true |
| codex | `<home>/.codex/prompts/spekk-dev-loop.md` | — (no project install) | true |
| copilot | — (no global path) | `<cwd>/.github/prompts/spekk-dev-loop.prompt.md` | true |

- **Native-skill harnesses (claude-code, opencode)** write the embedded
  `SKILL.md` **verbatim** — frontmatter and body byte-for-byte. Both use the
  same `name` + `description` frontmatter schema, so no transformation is
  needed. The file lands in the harness's native `skills/<name>/SKILL.md`
  location.
- **Command/prompt harnesses (cursor, codex, copilot)** write the skill **body
  with the YAML frontmatter stripped** (everything from the leading `---`
  through the closing `---` and the following blank line removed) as a single
  `/spekk-dev-loop` command / prompt file. These tools render the whole file as
  a prompt and cursor explicitly forbids frontmatter, so it must go. On these
  harnesses the dev loop becomes a **manually-invoked `/spekk-dev-loop`
  command** rather than an auto-invoked skill — understood and accepted.

### The write, generalized

After writing the agent shims, `Install()` resolves the descriptor's path for
the current scope (`globalPath(home)` or `projectPath(cwd)`). If that path is
`""`, no dev-loop file is written for this target+scope and the FS is never
touched. Otherwise it reads the embedded skill through the injected FS
(`Options.SkillFS` falling back to `install.DefaultSkillFS`, via the single
`skillEmbedPath` constant), applies the transform (verbatim, or
frontmatter-stripped when `strip`), creates the parent dir (`0o755`), writes the
file (`0o644`), and appends the path to the returned slice so the command prints
it like every other installed file. If a dev-loop file is due but no skill FS is
available, `Install()` fails loudly rather than silently skipping.

### Decisions (decided, not asked)

- **Slash-command-per-harness.** Native-skill harnesses get a verbatim skill;
  command/prompt harnesses get a frontmatter-stripped `/spekk-dev-loop` command.
  One embedded source, at most one transform (frontmatter strip).
- **Copilot is project-only.** Copilot personal/global prompts are IDE-managed;
  there is no standard global filesystem path for a personal prompt file.
  Rather than invent a fake `~/.copilot/prompts`, a global copilot install
  writes **no** dev-loop file (the agent shims still install). The dev-loop
  command lands only on `--project` at `<cwd>/.github/prompts/spekk-dev-loop.prompt.md`.
  This is the mirror image of `codex`, which has no project install and so is
  global-only.
- **Frontmatter strip is a shared helper.** A single `stripFrontmatter` function
  removes the leading `---` block and the following blank line. Verbatim targets
  don't call it; all three command targets share it. It is defensive: content
  that does not start with `---` is returned unchanged.
- **Paths and names are fixed.** Destinations are hard-coded per target. No new
  flags or config fields for users to point the dev loop somewhere else, and the
  file is always named for its harness's convention.
- **Overwrite silently.** Same idempotent behavior as the agent shims — a
  re-install overwrites the dev-loop file in place with no prompt or warning.
- **Embedded source is the single source of truth.** The verbatim source lives
  at `specs/install-spekk-dev-loop-skill/spekk-dev-loop-skill.md`; native
  targets get its exact bytes, command targets get exactly its body after the
  closing `---`.

### Temptations rejected

- **No general skill registry / arbitrary-skill installer.** This ships exactly
  the one `spekk-dev-loop` skill through the existing `spekk install` flow — not
  "install any skill to any target."
- **No configurable destinations.** No new fields for users to choose paths or
  names — paths are fixed per target.
- **No per-harness copies of the content.** One embedded source, transformed at
  most by frontmatter-stripping into the two classes. No forked skill bodies.
- **No class hierarchy.** Don't over-model the two classes — a path pair plus a
  `strip` bool on the existing per-target descriptor is enough. No interface, no
  enum, no registry object.
- **No fabricated paths.** No invented `~/.copilot/prompts` global location just
  to make the table symmetric — global copilot writes no dev-loop file.

## Assertions

See `assertions/` for what must be true.
