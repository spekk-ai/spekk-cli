---
id: go-builder-launcher
parent: golang-agents
created: 2026-04-05T12:16:00Z
priority: 1
status: in_progress
depends-on: go-prompt-resolver
branch: feature/golang-agents
---

# Go builder agent launcher

The Go builder launcher supports all builder modes: continuous, once, dry-run, interactive, confirm, and skill activation.

## Success Criteria

**Modes:**
- Default (no flags): continuous loop — get next assertion, launch claude, repeat
- `--once`: build one assertion then exit
- `--dry-run` / `-d`: display next assertion details without launching claude
- `--interactive` / `-i`: launch claude with `--system-prompt` (user drives session)
- `--confirm` / `-c`: prompt y/n/q before each build
- `spekk builder <skill>`: resolve and activate a builder skill

**Flag handling:**
- `--spec <id>` / `-s <id>`: filter assertions to specific spec
- `--assertion <id>`: target specific assertion
- `--help` / `-h`: display help

**Process management:**
- In continuous mode, SIGINT during a build interrupts claude but doesn't kill the parent
- SIGINT between builds exits gracefully
- In interactive mode, SIGINT is suppressed (claude handles Ctrl+C natively)
- Colored console output for status messages

**Build loop:**
- Calls parser to get next assertion
- Handles `complete`, `error`, and unexpected result types
- On parser error in continuous mode: log warning, retry after 5s
- On `complete` in continuous mode: wait 5s then check again
- 1s pause between iterations
