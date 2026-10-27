---
id: docs-incremental-reindex
parent: source-document-index
created: 2026-10-17T18:00:00Z
priority: 1
status: done
branch: feature/source-document-index
depends-on: docs-status-and-freshness
---

# The Document Part Re-Indexes Only What Changed

## Success criteria

- The index has a table `doc_files (path, size, mtime_ns, sha256)` with one row for each file the document corpus reads: each in-place document, each file under `raw/` and `formatted/`, and `.spekk/docs-paths`. It replaces the single `docs_fingerprint`.
- The dirty check is a stat walk compared with `doc_files`. It reads no file content when nothing changed.
- A file whose size or mtime differs from its row is hashed. If the sha256 is the same, only its row is updated, and no document is indexed again (a `touch` of 50 documents re-indexes none).
- Only the documents of changed, added, and deleted files are deleted and inserted again, with their tags and passages. A changed stub, raw file, or derived text re-indexes its raw document. So an edit of one document re-indexes one document.
- A change to `.spekk/docs-paths` or `raw/.gitignore`, a schema-version change, and `spekk index --force` rebuild the whole document part. When two in-place files share a document id, the re-index keeps the rule of a full scan: the first in path order is the document.
- After any sequence of edits, the `documents` and `passages` rows equal the rows of `spekk index --force`.

**Tests:** `internal/index/documents_test.go`: a touch with the same content re-indexes nothing; an edit, an add, and a delete each re-index one document; the rows after the edits equal a forced rebuild.
