---
id: prompt-command
parent: agent-install
created: 2026-06-11T00:00:00Z
priority: 1
status: done
---

# `spekk prompt <agent>` Prints the Resolved Agent Prompt

`spekk prompt <agent>` prints the layered-resolved prompt for `coach`, `builder`, or `observer` to stdout, suitable for piping or for a host-tool agent to consume.

## Success Criteria

- `spekk prompt coach` prints the coach prompt (resolved via the existing `PromptResolver` layering: local/global overrides, embedded base, global/local extensions) to stdout with no extra decoration.
- Works identically for `builder` and `observer`.
- Unknown agent name prints an error to stderr listing valid agents and exits 1.
- Missing agent argument prints usage to stderr and exits 1.
- `spekk prompt --help` shows usage.
- `spekk help` lists the `prompt` command.

**Tests:** `internal/cli/prompt_test.go` (resolution already covered); command wiring exercised via `cmd/spekk` dispatch.
