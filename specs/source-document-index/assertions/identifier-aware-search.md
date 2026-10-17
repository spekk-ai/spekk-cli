---
id: identifier-aware-search
parent: source-document-index
created: 2026-10-17T16:00:00Z
priority: 1
status: done
branch: feature/source-document-index
depends-on: index-holds-full-text
---

# Search Terms Match Code Identifiers Precisely

The query text an agent types becomes a safe FTS5 expression, and identifiers in documents match.

## Success criteria

- The query is split on whitespace. A part in double quotes (`"exact phrase"`) stays one term. Each term is written to FTS5 as a quoted string (internal `"` doubled), so no user text is read as FTS5 syntax: `NOT`, `AND`, `*`, `:`, `(`, and `-` are plain characters, and no query text gives an FTS5 syntax error.
- A term with no letter or digit is dropped. A query with no term left is an error: `search: the query has no words`.
- Terms are joined with `OR`. Results are ranked by `bm25(passages, 2.0, 1.0, 1.0)` (heading, body, terms).
- `snake_case`, `dotted.name`, and `kebab-case` terms match the same identifier in a document as an exact phrase, and do not match the parts in another order.
- An identifier with an internal case change in a document (`parseAllSpecs`, `HTTPServer`, `EnsureFresh`) is also indexed as its lowercased parts (`parse all specs`, `http server`, `ensure fresh`) in the `terms` column. A query for the whole identifier and a query for its parts both find it.

**Tests:** table tests for the query builder (quoting, operators as text, dropped terms, empty query), and one index test that `parseAllSpecs`, `parse specs`, `index_db`, and `foo.bar` each find the document that holds the identifier.
