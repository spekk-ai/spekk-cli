---
id: cli-show-json
parent: show-command
created: 2026-10-15T21:30:00Z
priority: 1
status: in_progress
---

# `spekk show --json` Prints the Data of the View

## Description

`spekk show --json` prints to stdout the same JSON object that `spekk show` puts into `template.html` (`showData` in `internal/show/show.go`). It writes no file and opens no browser.

A server that shows the spec tree of a repository needs this data. It reads the specs from a read-only bare repository, and it cannot read the HTML. With `--json`, the server and the explorer use one data model, and the server does not parse specs itself.

The assertion came from spekk-app. The id and all success criteria are the same.

## Success Criteria

- The output is one JSON object with the keys `projectName`, `specs`, and `assertions`. The objects are the same as the data in the HTML of `spekk show` for the same tree. A test compares the two for one fixture tree.
- `--json` writes nothing to `.spekk/` and does not call `openBrowser`.
- For a `specs/` directory with no specs, the output is `{"projectName": "<name>", "specs": [], "assertions": []}` and the exit code is 0.
- A parse error (for example a duplicate spec id) gives exit code 1 and the error text on stderr, as `spekk show` does today.
- `--json --cross-branch` prints the same object with the cross-branch fields that `spekk show --cross-branch` puts into its HTML: `crossBranch`, `degraded`, and `branches`, and on each item `foreign`, `crossBranch`, and `crossBranchSummary`. `--branch-filter` works with it.
- `--json --cross-branch` works in this git layout: `GIT_DIR` is a git directory whose `commondir` file contains the path of a bare repository, `GIT_WORK_TREE` holds only `specs/`, and `GIT_OBJECT_DIRECTORY` is a private directory with the bare repository as an alternate. A test runs it on such a fixture, with a conflict, and with the bare repository read-only.
- The order of `specs` and `assertions` is the order of `spekk show` since v1.34.0 (#268): priority, then id, then file path.
- `spekk show --help` lists the flag.

**Note:** The empty case must exit 0. A caller shows an empty state for it, and must not cache it as an error.

**Note:** `--json` with `--watch` is an error (exit 1). Watch mode serves a page, and `--json` prints once.
