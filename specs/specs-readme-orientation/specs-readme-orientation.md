---
id: specs-readme-orientation
created: 2026-07-15T00:00:00Z
priority: 2
branch: feat/specs-readme-orientation
---

# Orient Cold Readers to a `specs/` Tree

When a developer (or an AI agent) opens a project's `specs/` directory for the first time, nothing explains what the directory is, what the `assertions/` subfolders are, what the YAML frontmatter fields mean, or how to interact with the system. The tree must become self-explanatory to three audiences without running the CLI: a cold reader browsing on GitHub, an agent grepping files, and a developer who opens a single assertion `.md` and wonders what `status:` or `parent:` mean.

The solution is a single, deliberately-small piece:

**A CLI-managed `specs/README.md`.** `spekk init` writes (and re-generates) a committed README containing the concept model, a frontmatter field reference, and a schema version. The CLI owns exactly one region of that file, delimited by literal fence markers; everything outside the fence is human prose the CLI never touches. Regeneration is idempotent (byte-identical output when nothing changed) and safe on an already-initialized, hand-edited, or legacy-static README.

> **Removed:** an earlier version of this feature also instructed the coach and builder agents to prepend an HTML-comment frontmatter-explainer header to every newly authored spec/assertion `.md`. That was removed — it cluttered every spec file with boilerplate. `specs/README.md` is the single reference; generated files carry no inline header.

## Deliberately out of scope

- **No live status/index in the README.** `spekk status` remains the single authoritative live view. The managed block is concepts + a static schema reference only — it must contain zero per-spec state.
- **No `spekk sync` command.** Regenerating the fence during `spekk init` (and any existing regeneration path) is sufficient. Schema drift surfaces naturally as a diff the next time `init` runs, because the managed block is rendered purely from a current version constant.
- **No schema-migration framework.** A single current-version constant is enough; "regenerate from the current constant" is the whole mechanism.
- **No general templating/config/doc engine.** A couple of embedded Go string constants and marker-delimited string splitting are the bar. No mustache, no markdown/HTML parser.
- **No backfilling of existing committed spec files.** This feature governs the managed README only, not a sweep of the existing spec directories in this repo.

## Frontmatter reference this feature must document (authoritative — from `internal/parser`)

Spec files (`specs/<name>/<name>.md`): `id` (kebab-case, required), `created` (ISO-8601 `YYYY-MM-DDTHH:MM:SSZ`, required), `priority` (integer 1–3, required), `status` (default `not_started`), `branch` (default `main`).

Assertion files (`specs/<name>/assertions/<name>.md`): all of the above plus `parent` (required, the owning spec id), `depends-on` (assertion id that must be `done` first), `locked-by` (agent lock marker).

Valid `status` values: `not_started`, `in_progress`, `done`, `draft`, `failed`.
