# Spekk CLI 1.5.0 — The Go Rewrite

This release covers changes since v1.4.0. The headline: spekk is now a single static Go binary with no runtime dependencies.

## Complete Migration to Go (PR #70)

The entire CLI has been rewritten from Node.js to Go:

- Single static binary — no Node.js runtime, no `npm install`
- All commands ported: `init`, `next`, `coach`, `builder`, `observer`, `show`, `serve`, `sandbox`, `status`
- Cross-compiled binaries for macOS (amd64/arm64) and Linux (amd64/arm64) on every release
- Parser, agent launcher, spec explorer, WebSocket server, and sandbox orchestration all live in Go packages under `internal/`

## Embedded Skills and Prompts (PR #75)

Agent prompts and packaged skills are now embedded in the binary at compile time. Nothing is installed to disk — the binary is fully self-contained, and skills resolve from the embedded filesystem when no local or global override exists.

## Layered Skill Discovery (PR #62)

Coach and builder skills now resolve through layers, first match wins:

1. **Local**: `.spekk/skills/<agent>/` — project-specific skills
2. **Global**: `~/.spekk/skills/<agent>/` — your personal skills
3. **Package**: skills built into the binary

Skills can be invoked by filename or by frontmatter `id`, and legacy aliases (`spekk coach meeting`) keep working.

## Self-Update (PR #107)

New `spekk update` command upgrades the binary in place via the GitHub Releases API. No more manual downloads.

## Sandbox Improvements

- Deploy agent installs from GitHub releases instead of ad-hoc uploads (PR #65)
- Cloud-init provisioning restored and prepared for open-source (PR #86)
- New `--vpc` flag on `spekk sandbox create` (PR #60)

## Security Hardening (PR #108)

Eight vulnerabilities remediated across the agent launcher, sandbox, and serve packages, including SSH host key verification for sandboxes and WebSocket origin checks with session nonces.

## Documentation

- New documentation site built with Zensical (PR #67)
- Experimental branch workflow documented (PR #109)

## Other Changes

- Parser robustness: malformed spec files warn and skip instead of crashing (PR #55)
- CRLF parsing and Windows TTY spawn fixes (PR #54)
- Test isolation: tests use temp dirs instead of the project's `specs/` (PR #56)

## Upgrade

Download the latest binary from [GitHub Releases](https://github.com/spekk-ai/spekk-cli/releases) or build from source:

```bash
go install github.com/spekk-ai/spekk-cli/cmd/spekk@latest
```

From this version on, future upgrades are one command: `spekk update`.
