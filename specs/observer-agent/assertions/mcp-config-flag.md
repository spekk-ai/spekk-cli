---
id: mcp-config-flag
parent: observer-agent
created: 2026-10-15T23:00:00Z
priority: 2
status: done
branch: feat/observer-mcp-config
---

# `--mcp-config` Gives One Observer Run Its Own MCP Servers

A scheduled observer skill can need tools that no other session has, for example a browser. Claude Code loads such MCP servers with `--mcp-config <file>`. Before this change, `spekk observer` accepted `--mcp-config` without an error and did not give it to `claude`, so the run started without those tools and nothing said so.

**Tests:** internal/agent/observer_mcp_test.go

`claude --mcp-config` takes one or more values. When `--mcp-config <file>` comes before the message, `claude` reads the message as a second config file and fails. The observer must give the flag as one token, `--mcp-config=<file>`.

## Success Criteria

- `spekk observer [skill] --mcp-config <file>` gives `--mcp-config=<file>` to `claude` as one argument, and the message is still the last argument. This applies to `--headless` and to an interactive run, and to the default scan and to a skill run.
- Without `--mcp-config`, the arguments to `claude` do not change.
- `--mcp-config` with no value, or with a path that is not a regular file, stops the run with an error that names the flag and the path, and exits non-zero. `claude` does not start.
- `docs/cli-reference.md` lists the flag in the observer flag table.

## Verification

- `go test ./internal/agent` runs the observer with a fake `claude` that records its arguments, and checks the argument list with and without the flag. It also checks that a missing file and a missing value stop the run before the fake `claude` starts.
