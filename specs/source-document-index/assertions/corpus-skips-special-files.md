---
id: corpus-skips-special-files
parent: source-document-index
created: 2026-10-17T19:00:00Z
priority: 1
status: done
branch: feature/source-document-index
depends-on: hostile-documents-are-contained
---

# A Corpus Entry That Is Not a Regular File Is Not Read

A repository can commit a symlink to a device, and a working tree can hold a FIFO. Before this assertion, the manifest hashed each file under `raw/` and `formatted/`, so a `raw/` symlink to `/dev/zero` or a FIFO made every `spekk docs` command hang, and a `raw/` symlink to a file outside the repository was read.

## Success criteria

- The manifest of the corpus (`CorpusStat`) includes only regular files. It skips a FIFO, a device, a socket, and a symlink to one of them, under `raw/`, under `formatted/`, and in the in-place globs.
- The manifest skips a symlink under `raw/` or `formatted/` that resolves outside the repository, so nothing is read from its target.
- `FileSHA256` refuses a file that is not a regular file before it opens it, so no caller can block on a FIFO or read a device without end. The readers of in-place documents, stubs, and derived text refuse such a file the same way.
- With a `raw/` symlink to `/dev/zero`, a FIFO in `raw/`, a FIFO stub in `formatted/`, and a FIFO in `docs/`, `Refresh` returns, and the document rows equal a `--force` rebuild without those entries.

**Tests:** `internal/index/special_unix_test.go` (the four entries above; the test times out on the code before the fix).
