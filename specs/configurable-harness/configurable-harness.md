---
id: configurable-harness
created: 2026-09-04T00:00:00Z
priority: 2
---

# Configurable Agent Harness

## Overview

Spekk hardcodes the `claude` CLI as the agent harness at every point where it
spawns a coach, builder, or observer process. The binary name and the
Claude-specific flags (`--dangerously-skip-permissions`, `-p`, the not-found
guidance text) are literals scattered across the launch sites.

This spec makes the launch harness selectable so a user can run the same
spec-driven workflow under a different tool — starting with opencode. The
install side is already harness-agnostic: `spekk install --target opencode`
writes the prompts and skills into opencode's config dirs. What remains is the
runtime launch.

## Scope

In scope — the launch sites that spawn an interactive or headless agent, and
the observer cron entry:

- `internal/agent/launcher.go` — interactive coach/builder launch
- `internal/agent/builder.go` — builder launch
- `internal/agent/launcher_headless_unix.go` and `_windows.go` — headless launch
- `internal/agent/observer_cron.go` — the binary baked into the crontab entry

Out of scope — `internal/serve/serve.go`. `serve` bridges a WebSocket to Claude
Code's `stream-json` protocol in both directions; that is a wire protocol, not
a flag map, and no other harness speaks it today. `serve` stays Claude-only
(see `serve-stays-claude-only`).

## Harness selection

A harness is chosen by, in precedence order:

1. A `--harness` flag on the command
2. The `SPEKK_HARNESS` environment variable
3. The default, `claude-code`

## Design notes

A harness profile carries everything a launch site needs that used to be a
`"claude"` literal: the binary name, how a prompt is passed, how permissions
are skipped, how headless mode is requested, and the "not found — install X"
guidance shown when the binary is missing. Launch sites resolve a profile and
read those fields instead of embedding literals.

Supported harnesses: `claude-code` (default, alias `claude`), `opencode`,
`hermes`, `codex`, and `gemini`. Each harness has its own CLI shape — they share
no flag conventions — so every profile is independent.

## Flags are verified against the real CLI, never written from memory

The first opencode profile emitted flags opencode does not define (`--prompt`,
and `--auto` on the bare command), so the whole prompt was dropped and opencode
opened an empty TUI instead of running as the agent. The argv tests passed
because they only checked the profile's own output against itself. Every profile
must therefore be checked against its actual binary's `--help`
(`harness-flags-verified-against-cli`). For a harness whose binary is not yet
installed (`codex`, `gemini`), the builder installs it (or runs where it is
available) and reads the real `--help` before the profile can be done; an absent
binary leaves the assertion open rather than fabricated.
