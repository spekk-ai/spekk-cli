---
id: hostile-documents-are-contained
parent: source-document-index
created: 2026-10-17T18:00:00Z
priority: 1
status: not_started
branch: feature/source-document-index
depends-on: docs-add-converted-formats
---

# A Hostile Document Cannot Read Outside the Repository, Exhaust Memory, or Forge Output

## Success criteria

- A stub whose `path` is not under `raw/`, or whose `text` is not under `formatted/` (an absolute path, or `..` after `filepath.Clean`), is skipped with a warning and nothing is read from it. A path that resolves through a symlink to a place outside the repository is skipped the same way.
- An in-place document that is a symlink to a file outside the repository is not indexed.
- An XLSX cell reference above column 16,384 (`XFD`) fails the add with a clear message, never a panic.
- An XLSX part whose uncompressed size is above 64 MB, or a sheet text above 64 MB in total, fails the add with a clear message. Each part is read through a size limit, so a wrong size in the zip header does not help.
- `pdftotext` and `pandoc` run with a timeout of 60 s and an output cap of 64 MB; on either, the add of that file fails with a message that names the limit.
- The begin and end markers of untrusted text in `docs show` hold a random value that is new for each call.

**Tests:** `internal/docs/corpus_test.go` (a stub with `../` and a symlink out of the root are not indexed) and `internal/docs/add_test.go` (the column bound, a small XLSX bomb).
