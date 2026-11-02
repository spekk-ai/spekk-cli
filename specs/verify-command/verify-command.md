---
id: verify-command
created: 2026-11-02T21:30:00Z
priority: 1
---

# `spekk verify` - the Truth of Each Property at HEAD

## Problem

A stale spec is worse than no spec: the agent skips the code and acts on a false belief. Spekk must therefore say, for each property, whether it is true at HEAD and how strongly it is checked. Code shows what the system does, but not whether that is intended, so the evidence must come from checks that name the property they check.

`spekk verify` runs the test commands of the project, reads their JUnit XML, maps each result to its property through the evidence markers (`evidence-markers`), and records the truth of each property for the HEAD commit in `.spekk/index.db`.

## Configuration

The test commands are in `spekk.toml` at the project root (the directory that holds `specs/`):

```toml
[[verify.run]]
command = "go test ./... -json 2>&1 | go-junit-report -parser gojson > .spekk/junit/go.xml"
junit = ".spekk/junit/go.xml"

[[verify.run]]
command = "pytest --junitxml=.spekk/junit/py.xml"
junit = ".spekk/junit/py.xml"
```

`strictness-config` adds the `[strict]` table to the same file.

## Truth at HEAD

| Value | Meaning |
|-------|---------|
| `verified` | The property has checks. No check failed at HEAD, and at least one passed. |
| `failing` | A check of the property failed or gave an error at HEAD. |
| `unchecked` | No check result for the property at HEAD. This includes prose only. |
| `unconfirmed` | Prose only, and the anchored code changed after the last confirmation. |

## Decisions

- Each property has two values: the workflow value (v1 `status`) and the truth at HEAD, which spekk calculates again at each commit (William, 2026-11-01).
- `spekk verify` reads JUnit XML. pytest, vitest, cargo test and Go (with a converter) can all write it.
- The level of a property is its strongest passing evidence. Spekk does not trust a declared level.

## Open choices

- **Confirmation.** The last confirmation of a prose property is the last commit that changed its assertion file. A person or an agent who reads the code and the property again confirms it with a commit to the file. This needs no new state.
- **Anchored code.** The anchored code is each file that holds a comment anchor of the property, and each front-matter anchor. A change anywhere in such a file counts.
- **Checks with no result.** A property with markers but no result at HEAD is `unchecked`, not `failing`. This is true when no run happened at HEAD, when its tests were skipped, and after a force rebuild of the index. Nothing checked it at HEAD.
- **Uncommitted changes.** A run on a work tree with uncommitted changes counts for HEAD. The run is recorded as `dirty`, and the display says so.
- **The latest run wins.** When there are two or more runs for one commit, the results of the latest run are the truth for that commit. Results are not merged across runs.
- **JUnit files.** A `junit` value can be a glob. A JUnit file that was not written during the run is not read, so a stale file never gives evidence.
- **Exit code.** `spekk verify` exits 1 when a command exits non-zero, when a JUnit file is missing, or when a property is `failing`. Strictness can add more failures.
- **Index rebuilds.** A content rebuild of the index keeps the verify tables. A force rebuild or a schema-version rebuild drops every table (`force-flag-rebuilds-from-scratch`), and the properties show `unchecked` until the next `spekk verify`.

## Assertions

See `assertions/` for what must be true.
