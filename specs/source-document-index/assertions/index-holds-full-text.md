---
id: index-holds-full-text
parent: source-document-index
created: 2026-10-17T16:00:00Z
priority: 1
status: not_started
branch: feature/source-document-index
depends-on: passage-chunking
---

# The Index Holds the Full Text of Specs and Assertions

`spekk index` (and `EnsureFresh`) writes the bodies of specs and assertions into a full-text table next to the existing metadata tables.

## Success criteria

- New tables in `.spekk/index.db`:
  - `documents(kind, id, path, text_path, title, type, created, summary, sha256, state)`, primary key `(kind, id)`. `kind` is one of `spec`, `assertion`, `doc`, `raw`. `path` is relative to the repository root. `text_path` is the file the line numbers refer to.
  - `document_tags(kind, doc_id, tag)`.
  - `passages`, an FTS5 table with columns `meta` (title, summary, and tags of the document), `heading`, `body`, `terms` (identifier parts, see `identifier-aware-search`), and unindexed `kind`, `doc_id`, `start_line`, `end_line`. Tokenizer: `porter unicode61 remove_diacritics 2`.
- Each spec and each assertion has one `documents` row (summary = its title, `created` from frontmatter) and its passages from `passage-chunking`, with line numbers in its own file.
- The existing tables and their rows do not change. `schemaVersion` goes to 5, so an index that version 4 built rebuilds on first use.
- `spekk index` prints `Indexed N specs, N assertions, N observations, N documents (N passages).`, where documents counts kinds `doc` and `raw`.
- `spekk query "SELECT kind, id FROM documents"` and a `passages MATCH` query work through the read-only connection.

**Tests:** extend the existing build test: a spec tree in a temp dir gives `documents` rows for each spec and assertion, and a `MATCH` on a word that only one assertion body holds returns that assertion with correct line numbers.
