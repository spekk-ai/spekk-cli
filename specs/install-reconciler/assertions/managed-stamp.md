---
id: managed-stamp
parent: install-reconciler
created: 2026-07-27T00:00:00Z
priority: 1
status: done
branch: feat/install-reconciler
---

# A Managed File Carries a Stamp with a Body Hash

## Description

Every file that `spekk install` writes gets a stamp. The stamp is a single
trailing marker line. A trailing marker works for every target, because the
codex target writes files with no frontmatter. The stamp marks the file as
spekk-managed and holds a hash of the body.

## Success Criteria

- A `StampContent(body []byte) []byte` function appends a trailing marker line to
  the body. The marker has the form `<!-- spekk:managed hash=<hex> -->`. The
  `<hex>` value is the SHA-256 hash of the body, in lowercase hex.
- A `ParseStamp(content []byte) (body []byte, hash string, managed bool)`
  function reads a stamped file. For stamped content it returns the body without
  the marker, the hash from the marker, and `managed = true`. For content with no
  marker it returns `managed = false`.
- `ParseStamp(StampContent(b))` returns the original body `b`, the correct hash,
  and `managed = true`, for any body `b`.
- A helper reports whether the on-disk body agrees with the stamp: it hashes the
  parsed body and compares the result with the parsed hash. It returns true when
  the two agree.
- The functions live in `internal/install` and have unit tests.
