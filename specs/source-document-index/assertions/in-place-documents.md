---
id: in-place-documents
parent: source-document-index
created: 2026-10-17T16:00:00Z
priority: 2
status: done
branch: feature/source-document-index
depends-on: index-holds-full-text
---

# Prose Already in the Repository Is Indexed in Place

## Success criteria

- With no `.spekk/docs-paths`, the include globs are `docs/**/*.md`, `docs/**/*.txt`, and `*.md` (files at the repository root only).
- `.spekk/docs-paths` replaces the defaults: one glob for each line; a line that starts with `!` is an exclude; `#` starts a comment; blank lines are ignored. `**` matches zero or more directories, `*` and `?` match inside one path segment. Paths use `/` on every platform.
- Files under `specs/`, `raw/`, `formatted/`, `.spekk/`, and `.git/` are never in-place documents, whatever the globs say.
- Only `.md`, `.markdown`, and `.txt` files are in-place documents. Other matched files are skipped without a warning.
- Each in-place document has kind `doc`, id = kebab-case of its path without the extension (`docs/advanced/index.md` -> `docs-advanced-index`, `README.md` -> `readme`).
- Frontmatter on the file is optional. When present, `title`, `summary`, `type`, `tags`, and `created` are read. Without a `title`, the first heading is the title, else the file name. Without a `summary`, the first paragraph of the body (cut to 300 characters) is the summary.
- Two documents with the same id: the first in path order is indexed, and the build reports a warning that names both paths.
- A stub in `formatted/` that does not parse is skipped with a warning (from `spekk index` and `spekk validate`), so one broken stub does not stop every command.

**Tests:** `internal/docs/corpus_test.go`: the glob matcher (`**` at the start, middle, and end, `!` excludes, root-only `*.md`), and the id rule.
