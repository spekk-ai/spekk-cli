---
id: docs-refresh-never-blocks
parent: source-document-index
created: 2026-10-17T18:00:00Z
priority: 1
status: not_started
branch: feature/source-document-index
depends-on: docs-incremental-reindex
---

# A Query Never Waits for a Large Re-Index, and Parallel Queries Never Fail

## Success criteria

- The dirty check (stat walk and manifest read) is measured: under 20 ms for 5,000 files on the sandbox, or the measured number and the reason are in the PR.
- When the changed files are at most a threshold (taken from a measurement of the cost of one document, so that the inline work is about 100 ms or less), the query updates the index inline and answers from the fresh index.
- Above the threshold, the query answers from the current index at once. `docs search --json` then has `"stale": {"changed": N, "reindexing": true}`, and text output has one line that says the index is being updated. The query starts one background `spekk index` that holds `.spekk/index.lock` (created with O_EXCL). A second query does not start a second one while the lock is held. A lock older than 10 minutes is removed.
- With no index at all, the first query builds it inline (there is nothing to answer from).
- The index uses `journal_mode=WAL` and a busy timeout. Three parallel `spekk docs search` runs after an edit all exit 0.

**Tests:** `internal/index/documents_test.go`: parallel refreshes of the same edit all succeed and index once. The background path is checked in the walkthrough (it starts a process).
