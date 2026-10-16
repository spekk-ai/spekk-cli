---
id: show-help-exits-zero
parent: show-command
created: 2026-10-15T00:00:00Z
priority: 1
status: done
---

# `spekk show --help` Prints Help and Exits 0

## Description

`spekk show --help` must behave like `--help` on the other subcommands: it prints usage and exits 0. Before this fix, `show` did not know the flag, so `spekk show --help` wrote the explorer and opened a browser, and `spekk show --watch --help` started the watch server.

**Tests:** cmd/spekk/show_test.go (`TestExecShow_Help_ExitZero`)

## Success Criteria

- `spekk show --help` and `spekk show -h` print a usage block for `show` to stdout that names `--watch`, `--cross-branch`, `--branch-filter`, and `--help`, and exit 0.
- With `--help`, `show` does not parse specs, does not write `.spekk/index.html`, does not open a browser, and does not start a server. This is true also when `--help` comes with other flags, for example `spekk show --watch --help`.
- `spekk show` without `--help` is unchanged.
