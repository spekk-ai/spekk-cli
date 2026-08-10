---
id: parse-spec-from-ref
parent: show-cross-branch
created: 2026-06-15T12:00:00Z
priority: 1
status: done
---

# Spec/Assertion Files Are Parsed From Git Refs Using the Existing Parser

## Description

To know what a spec/assertion looks like on another branch, file content is read
out of a git ref and turned into Spec/Assertion structs using the **existing**
`internal/parser` logic. Git answers the merge/diff questions; the existing
parser answers the what's-in-the-file questions. No duplicate parsing logic is
introduced.

## Success Criteria

- File content for a spec/assertion is read from a ref via `git show <ref>:<path>`
  (or the equivalent against a merged tree) — never by checking the branch out.
- That content is parsed into the same `Spec` / `Assertion` structs produced by
  `internal/parser` for working-tree files, reusing the existing
  frontmatter/title/status parsing rather than re-implementing it.
- A content-based parse entry point is available from `internal/parser` (the
  existing per-file parse logic that takes file content is reused/exposed as
  needed) so callers can parse a string of file content without it existing on
  disk.
- Parsing is robust to a file that is missing on a given ref (returns a clear
  "not present" signal rather than erroring) and to malformed content on a ref
  (the error is contained to that file/branch, not the whole explorer).
