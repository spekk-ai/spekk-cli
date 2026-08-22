---
id: parser-collects-warnings
parent: parser-warnings-belong-to-the-caller
created: 2026-08-22T15:00:00Z
priority: 1
branch: fix/parser-warning-volume
status: not_started
---

# `ParseAllSpecs` Returns Its Warnings and Writes Nothing

The parser stops deciding what a user sees. It records what it skipped on the result it returns.

## Success criteria

- `ParseResult` gains a `Warnings []string` field.
- Every `fmt.Fprintf(os.Stderr, ...)` call in `internal/parser/parser.go` is deleted. After this change the package writes to `os.Stderr` from nowhere, and a test proves it.
- Seven of the eight warnings keep their present text and are appended to `Warnings` in the order the walk meets them. The walk sorts its entries, so the order is deterministic.
- **The eighth is deleted, not moved.** `Warning: Spec %s/ has no assertions/ directory — skipping.` reports a skip that does not occur: the spec is appended before the check, and the `continue` only ends the assertion scan. A spec directory with no `assertions/` directory is a normal, correctly parsed spec with no assertions, and `spekk status` already shows it as `0/0 assertions complete`. Its `continue` stays; only the message goes.
- A shared formatter renders the one-line summary, so the five callers do not each spell it out:
  - `func (r *ParseResult) WarningSummary() string` returns `""` when `Warnings` is empty.
  - Otherwise it returns exactly:
    `Warning: 20 spec files skipped and missing from the queue. Run "spekk validate" for detail.`
  - The count is `len(r.Warnings)`, and the noun agrees: `1 spec file skipped`, `2 spec files skipped`.
- `ParseAllSpecs` does not call `WarningSummary`. The caller decides whether to print, which is the whole point of the change.

**Note:** This is a pure move of the warnings from stderr onto the result. Do not change which files the parser skips, and do not change the wording of the seven that stay. A behavior change hidden inside a plumbing change is hard to review.

**Tests:** `internal/parser/` — a fixture tree with a malformed assertion, a malformed spec file, a duplicate assertion id, and a spec directory holding assertions but no main spec file returns one `Warnings` entry for each, and writes nothing to a captured `os.Stderr`; a spec directory with no `assertions/` directory produces **no** warning and still appears in `Specs`; `WarningSummary()` returns `""` for an empty slice, the singular form for one, and the plural form with the right count for many.
