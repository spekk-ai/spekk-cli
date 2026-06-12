---
icon: lucide/megaphone
---

# Release Notes

What's new in each version of Spekk CLI.

---

## [1.4.0 -- Layered Prompts, Sandboxes, WebSocket Server](RELEASE-NOTES-1.4.0.md)

Agent prompts can now be customized globally (`~/.config/spekk/`) or per-project (`.spekk/`) with extend and override files. New sandbox commands for cloud agent environments. WebSocket server for real-time integrations.

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
