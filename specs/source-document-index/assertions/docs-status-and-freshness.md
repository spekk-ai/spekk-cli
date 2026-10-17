---
id: docs-status-and-freshness
parent: source-document-index
created: 2026-10-17T16:00:00Z
priority: 1
status: done
branch: feature/source-document-index
depends-on: docs-add-text-formats
---

# Changed Documents Go Stale, and the Index Follows the Corpus

## Success criteria

- Each `raw` document has a state, written to `documents.state` at build time: `ok`; `stale` (the sha256 of the raw file differs from the stub's `sha256`); `no_summary` (the summary is empty); `missing_raw` (the stub's `path` does not exist and is not in `raw/.gitignore`); `pointer` (the path is in `raw/.gitignore` and the file is absent). `stale` wins over `no_summary`. Every other kind is `ok`.
- A file in `raw/` with no stub that points at it is reported as `no_stub` by `spekk docs status`. It is not indexed.
- `spekk docs status` prints one line for each document that is not `ok` (`<state>  <id>  <path>  <what to run>`), then a count. With nothing to report it prints `All N source documents are OK.` It exits 0.
- `EnsureFresh` rebuilds two parts on their own. The spec part (specs, assertions, their passages, observations) rebuilds when a file or directory under `specs/` is newer than `index.db` (a directory counts, so a deleted spec file is seen) or the observation refs changed. The document part (kinds `doc` and `raw`) rebuilds when the corpus fingerprint changed: the path, size, and mtime of each in-place document, of each file under `raw/` and `formatted/`, and of `.spekk/docs-paths`, stored in `index_meta`. So an edit, an added file, and a deleted file each cause a rebuild of their part, a file that no glob selects causes none, and a spec edit does not index the documents again. The schema-version rule stays.
- `spekk index --force` on a fresh clone gives the same `documents` and `passages` rows as the index the author had (no state lives only in the database).

**Tests:** the state rules (each state from a temp corpus, and `stale` over `no_summary`), and a freshness test where a deleted document causes a rebuild and an unrelated file does not.
