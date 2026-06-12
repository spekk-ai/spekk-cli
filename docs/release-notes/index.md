---
icon: lucide/megaphone
---

# Release Notes

What's new in each version of Spekk CLI.

---

## [1.7.1 -- Update Permission Fix](RELEASE-NOTES-1.7.1.md)

`spekk update` now fails fast with clear guidance when the install directory isn't writable (e.g. a sudo install to `/usr/local/bin`), instead of a raw permission error after starting the download.

## [1.7.0 -- XDG-Compliant Config Directory](RELEASE-NOTES-1.7.0.md)

The global config directory moves from `~/.spekk` to `~/.config/spekk` (honoring `$XDG_CONFIG_HOME`), with automatic migration of existing directories. Platform support clarified: macOS and Linux.

## [1.6.0 -- Use Spekk from Any Coding Assistant](RELEASE-NOTES-1.6.0.md)

New `spekk install` registers the spekk agents as subagents in Claude Code, Cursor, Copilot, OpenCode, or Codex. New `spekk prompt` and `spekk skill` commands let agents in any harness fetch their instructions and skills on demand.

## [1.5.0 -- The Go Rewrite](RELEASE-NOTES-1.5.0.md)

Spekk is now a single static Go binary — no Node.js runtime required. Skills and prompts are embedded in the binary, `spekk update` self-updates via GitHub Releases, and eight security vulnerabilities were remediated.

## [1.4.0 -- Layered Prompts, Sandboxes, WebSocket Server](RELEASE-NOTES-1.4.0.md)

Agent prompts can now be customized globally (`~/.spekk/`) or per-project (`.spekk/`) with extend and override files. New sandbox commands for cloud agent environments. WebSocket server for real-time integrations.

## [1.3.0 -- Cloud Sandboxes](RELEASE-NOTES-1.3.0.md)

New `spekk sandbox` command for provisioning and managing DigitalOcean droplet-based agent sandboxes.

## [1.2.4 -- Live Reload + Searchbar](RELEASE-NOTES-1.2.4.md)

`spekk show --watch` for live-reloading the spec explorer as specs change. Searchbar filtering in the explorer UI.

## [1.2.1 -- Dependency Visualization](RELEASE-NOTES-1.2.1.md)

`spekk show` now renders an interactive metro map of dependency trees for each branch.

## [1.2.0 -- Coordinator Skill & Skills Architecture](RELEASE-NOTES-1.2.0.md)

Coordinator skill for dependency-aware work planning. Skills converted from JS classes to markdown files.

## [1.1.0 -- Builder CLI Flags](RELEASE-NOTES-1.1.0.md)

Builder now loops continuously by default. New flags: `--once`, `--dry-run`, `--spec`, `--assertion`, `--confirm`, `--interactive`.
