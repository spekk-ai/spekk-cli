---
id: derived-only-invariant
parent: observation-index
created: 2026-07-26T12:00:00Z
priority: 1
status: done
---

# Every Index Table Is Rebuildable From Plaintext or Safe to Lose; Prompts Get SELECT-Only Access; Writes Are Go-Only

## Description

The SQLite index is a derived, ephemeral layer — never a source of truth.
This invariant is stated explicitly, holds for every table (existing and
observation tables alike), and is structurally enforced: agents can only
SELECT, and only Go code paths write.

## Success Criteria

- The invariant is stated verbatim in `internal/index` package documentation
  and in the parent spec: **every table in `.spekk/index.db` is either
  rebuildable from plaintext in the repo/branch set, or safe to lose.**
- Deleting `.spekk/index.db` and re-running `spekk index` reproduces
  equivalent contents for all tables from the repo and its visible refs — a
  test demonstrates this round-trip for the observation tables.
- No lifecycle state exists only in SQLite: nothing in the observer flow
  (dedup, announce eligibility, digest rendering) is derivable *only* from
  the database — each is re-derivable from branches + frontmatter, with the
  index serving as an accelerator.
- Prompt/agent access is SELECT-only via `spekk query` (read-only connection
  and statement validation per `specs/sqlite-index/`); the observation tables
  introduce no write-capable surface reachable from a prompt.
- All writes to the database happen through Go code in `internal/index` —
  no skill, prompt, or shell instruction anywhere in the repo tells an agent
  to write to `.spekk/index.db` directly.
- `.spekk/index.db` remains gitignored.

**Note:** this is the load-bearing lesson from the production failure — state
that exists only in a derived or untracked artifact will eventually be lost
or self-referential. The index may cache, accelerate, and join; it may never
remember.

**Tests:** internal/index/observation_test.go (TestObservationIndexRoundTrip)
