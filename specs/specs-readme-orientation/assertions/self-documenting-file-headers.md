---
id: self-documenting-file-headers
parent: specs-readme-orientation
created: 2026-07-15T00:00:00Z
priority: 2
status: not_started
branch: feat/specs-readme-orientation
depends-on: managed-readme-block
---

# Newly Authored Spec/Assertion Files Carry an Inline Frontmatter Header

A developer opening a single assertion `.md` should understand what `status:`
and `parent:` mean without leaving the file. Because spec and assertion files in
this system are authored by the **agents following their prompts** (the coach
authors specs/assertions; there is no CLI command that scaffolds a spec or
assertion `.md`), the leanest home for the header template is the agent prompt
files themselves. This assertion adds a fixed HTML-comment header template to
the coach prompt (and the builder prompt, which also authors assertion files)
so every newly generated file carries it.

## Design decision (home for part 2)

Rejected: (a) a new CLI scaffolding command — adds a command that competes with
the agent authoring workflow to solve a documentation problem; (c) a
`spekk`-exposed template string with no caller — nothing consumes it. Chosen:
**(b) prompt-embedded template.** The agents already own file creation, so the
header ships wherever files are actually written, with zero new CLI surface.

## Success Criteria

- **Coach prompt updated.** `specs/coach-agent/coach.prompt.md` instructs the
  coach that every spec file and every assertion file it authors must begin with
  an HTML-comment header, placed immediately after the closing `---` of the
  frontmatter, before the `# ` H1.
- **Builder prompt updated.** `specs/builder-agent/builder.prompt.md` carries the
  same instruction for any assertion file the builder creates, so the two agents
  stay consistent.
- **Header content (fields).** The template briefly explains each frontmatter
  field on its own line — spec header covers `id, created, priority, status,
  branch`; assertion header additionally covers `parent, depends-on, locked-by`
  — with one terse phrase each (e.g. `status: not_started | in_progress | done |
  draft | failed`). It is intentionally terse; `specs/README.md` remains the
  fuller reference (single source of truth), so the header does not restate the
  full docs.
- **Pointer line.** The header includes a one-line pointer such as
  `See specs/README.md for the full concept + frontmatter reference.`
- **HTML comment form.** The header is wrapped in `<!-- ... -->` so it renders
  invisibly on GitHub but is visible in raw view and editors. It is provided as
  a literal template block in the prompt for the agent to copy, not described
  abstractly.
- **No backfill.** This assertion changes only the prompts (what is generated
  going forward). It must NOT sweep or rewrite the existing committed spec files
  in this repo.

**Note:** This is prompt-driven ("best effort," authored by an LLM), not a
byte-exact CLI guarantee. That is an accepted trade-off for a documentation
header that renders invisibly and changes rarely — see the open question in the
coach summary about whether a CLI-guaranteed header is ever wanted. Keeping the
field docs terse and pointing to `specs/README.md` avoids duplicating (and
having to keep in sync) the fuller reference in two authoritative places.

## Tests

Content assertions on the prompt files: the coach prompt contains the literal
HTML-comment header template for both spec and assertion files, the header names
every frontmatter field, and it contains the `specs/README.md` pointer line;
the builder prompt contains the assertion header template. (No Go runtime code
path here — the check is on the committed prompt text.)
