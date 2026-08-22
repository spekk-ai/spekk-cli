---
id: commands-print-one-summary-line
parent: parser-warnings-belong-to-the-caller
created: 2026-08-22T15:00:00Z
priority: 1
branch: fix/parser-warning-volume
depends-on: parser-collects-warnings
status: done
---

# `next`, `list`, `status`, and `show` Print One Line, Not One Per File

A command whose job is to answer a question says that something is missing, and points at the command whose job is the detail.

## Success criteria

- Each of these call sites prints `result.WarningSummary()` to **stderr** when it is not empty, and prints nothing when it is:
  - `cmd/spekk/main.go:167` (`next`)
  - `cmd/spekk/main.go:345` (`list`)
  - `internal/status/status.go:29`
  - `internal/show/show.go:129`
- `internal/index/index.go:223` prints **nothing**. The index rebuild is not a user-facing command, and this is what ends the doubling: `spekk next` parses twice, and only the answering parse now reports.
- `internal/show/watch.go:188` prints nothing. It re-parses on a loop, so a summary there would repeat on every tick.
- The summary goes to stderr, never stdout. `spekk list --json`, `--csv`, and `--tsv` must stay machine-readable, and `spekk next` must stay parseable as JSON.

### Measured result
On the 20-malformed-assertion tree in the parent spec:

| Command | stderr before | after (cold) | after (warm) |
|---|---|---|---|
| `spekk next` | 40 | **1** | **1** |
| `spekk list` | 20 | **1** | **1** |
| `spekk status` | 20 | **1** | **1** |

The cold and warm figures are equal, because the doubling is gone.

**Note:** `spekk validate` prints no summary. It already reports each problem as a failure line on stdout with the file and the exact fault, which is the detail the summary points at.

**Tests:** `cmd/spekk/` and `internal/status/` — over a fixture tree with several skipped files, each command writes exactly one warning line to stderr and its normal output to stdout; `spekk list --json` stdout parses as JSON with the warning present on stderr; a clean tree writes nothing to stderr.
