---
id: managed-readme-block
parent: specs-readme-orientation
created: 2026-07-15T00:00:00Z
priority: 1
status: not_started
branch: feat/specs-readme-orientation
---

# `spekk init` Writes a Fenced, CLI-Managed Block in `specs/README.md`

`spekk init` produces a `specs/README.md` whose body contains one CLI-owned
region delimited by literal fence markers. This assertion covers the fresh-init
path (no README yet) and defines the marker contract and the rendered content;
the in-place regeneration behavior is `idempotent-regeneration`, and existing/legacy/corrupt-file handling is `fence-state-handling`.

## Success Criteria

- **Exact markers.** The managed region begins with a line containing exactly
  `<!-- spekk:managed:begin -->` and ends with a line containing exactly
  `<!-- spekk:managed:end -->`. These two literal strings are the contract; they
  are defined as named constants (one begin, one end), not scattered string
  literals.
- **Fresh init.** When no `specs/README.md` exists, `spekk init` writes a README
  consisting of human-facing intro prose (outside the fence) followed by the
  managed region. The file ends with exactly one trailing newline.
- **Managed content (concepts).** Between the markers the block documents the
  concept model in prose: what `specs/` is, that each spec is a folder with a
  `<name>.md` describing what must be true, that `assertions/` holds small
  testable steps, and the `spekk init` → `spekk coach` → `spekk builder` /
  `spekk next` / `spekk status` command loop.
- **Managed content (frontmatter reference).** The block documents every
  frontmatter field for both spec and assertion files, matching
  `internal/parser` exactly: spec fields `id, created, priority, status,
  branch`; assertion fields add `parent, depends-on, locked-by`. It states the
  required-vs-optional status, the formats (`id` kebab-case; `created` ISO-8601
  `YYYY-MM-DDTHH:MM:SSZ`; `priority` integer 1–3; `branch` defaults to `main`),
  and the full valid `status` set: `not_started, in_progress, done, draft,
  failed`.
- **Schema version.** The block renders a `spekk_schema_version` (a single
  current-version constant) inside the fence. Its purpose is drift detection via
  diff — not a separate command.
- **No per-spec state.** The rendered block is a pure function of the schema
  version constant. It contains no spec names, no counts, no statuses, no
  timestamps, no host- or repo-specific data — nothing that would make two
  different projects' managed blocks differ at the same schema version.
- **Content lives in embedded Go string constant(s)**, in the same spirit as the
  existing `specsReadme` constant in `cmd/spekk/main.go` — not assembled by a
  templating engine.

**Note:** Because the whole block is rendered from the version constant, no
separate "compare versions" logic is needed. Bumping the constant changes the
rendered bytes, so the next `spekk init` naturally produces a diff. This is the
entire schema-drift mechanism; a `spekk sync` command is out of scope.

## Tests

Go test on the render function: asserts both exact marker strings are present,
that every frontmatter field name and each valid status value appears in the
block, that the `spekk_schema_version` line appears, and that rendering twice
returns byte-identical output (purity). A fresh-init CLI smoke test (empty temp
dir) confirms `specs/README.md` is created containing the managed region and a
single trailing newline.
