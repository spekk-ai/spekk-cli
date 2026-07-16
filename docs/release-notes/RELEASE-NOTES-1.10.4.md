# Spekk CLI 1.10.4 — `spekk list` + Coach Precision

This release adds a dedicated enumeration subcommand and sharpens the coach's spec-writing guidance.

## `spekk list`

A new subcommand for filtered spec and assertion enumeration. It replaces the pattern of calling `spekk next --all` and grepping the output.

**Status filtering:**

```bash
spekk list --status draft
spekk list --status in_progress
spekk list --status not_started
```

**Output formats:** `--json`, `--tsv`, `--csv`, and `--long` (adds a FILE column).

**Token impact:** `spekk list --status draft` returns ~5.8K tokens versus 92K+ for a full flat-file scan — a 16× reduction for enumeration queries. The coach uses it for session orientation, spec lookup, and post-write validation instead of loading the whole spec tree.

## Coach prompt improvements

- **Filtered enumeration** via `spekk list` in session orientation, spec lookup, and post-write validation.
- **Encoding precision for non-obvious constraints.** When a success criterion involves a behavioral constraint an implementer could get wrong — output format, ordering semantics, edge-case behavior, library defaults — the coach now states the exact constraint in the criterion rather than the category, and asks one targeted question when it isn't clear from the request. Examples: "sorted alphabetically; equal items preserve input order (stable sort)", "RFC 4180-compliant CSV; each row ends with CRLF".
- Refined the "one question when truly stuck" trigger and success-criteria output-format examples.

## Upgrade

```bash
spekk update
```
