---
id: deduplicate-parse-flags
parent: builder-cli-flags
created: 2026-02-25T00:14:00Z
priority: 1
status: done
---

# CLI Commands Share Argument Parsing

## Description

CLI commands use `internal/cli.ParseFlags` to recognize flag names and values. Commands can inspect argument errors and presence counts without parsing the argument list again.

## Success Criteria

- Builder and parser commands use the shared argument parser.
- Parsed results report the count for each recognized flag, including aliases and occurrences with missing values.
- Parsed results report unknown arguments and missing or empty string values. A later valid occurrence cannot hide an earlier missing value.
- Negative numeric values reach command-specific value validation.
- `spekk list` and `spekk observer install-cron` reject reported argument errors, including unsupported `--flag=value` syntax. Both use the shared parser without a separate argument scan.
- Existing flag values and aliases retain their behavior for other commands.

**Tests:** `internal/cli/flags_test.go`, `internal/agent/observer_cron_test.go`, `cmd/spekk/list_test.go`
