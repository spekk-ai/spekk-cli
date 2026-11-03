---
id: verify-runs-configured-commands
parent: verify-command
created: 2026-11-02T21:30:00Z
priority: 1
status: done
branch: feat/v2-verify-command
---

# `spekk verify` Reads Only JUnit Results That Its Own Run Wrote

## Description

`spekk verify` runs each `[[verify.run]]` command of `spekk.toml`, in file order, with `sh -c` from the project root. After each command it reads the JUnit XML files that its `junit` value names.

## Success criteria

- Each command runs once, in file order. A command that exits non-zero does not stop the run: spekk still reads its JUnit files, and runs the next command.
- A JUnit file is read only if its modification time is at or after the start of the run. A file from an earlier run is not read. Verify prints `<path>: not written by this run` to stderr and exits 1.
- A `junit` value that matches no file is an error of the same kind: a message on stderr and exit 1.
- A command that exits non-zero makes `spekk verify` exit 1 after the run. Its results are still recorded.
- With no `spekk.toml`, or with no `[[verify.run]]` entry, `spekk verify` prints an error that tells how to configure a run, and exits 1. It runs nothing.
- The output of each command goes to stderr, so that stdout holds only the verify report.
- `spekk verify` is listed in `spekk help`, and `spekk verify --help` prints its usage and exits 0 with no run.
