---
id: help-and-readme-document-install
parent: skill-install-system
created: 2026-05-22T12:00:00Z
priority: 3
status: in_progress
locked-by: builder-paris-macbook-pro-75928-1779480315
depends-on: install-command-parses-args
branch: feature/skill-install-system
---

# Help Output and README Document the Install System

## Description

The new commands appear in `spekk help` and have their own per-command help. `README.md` gains an "Installing Skills" section.

## Success Criteria

- `spekk help` lists `install`, `uninstall`, and `skills` in the commands table
- `spekk install --help` documents `<agent>`, `<skill>`, `--global`, `--local`, `--source`, `--force`, `--list`
- `spekk uninstall --help` documents `<agent>`, `<skill>`, `--global`, `--local`
- `spekk skills --help` documents `list`
- `README.md` has an "Installing Skills" section that shows the four common invocations: install from registry, install globally, install from `--source`, uninstall
- The README section mentions the official registry repo path (`github.com/spekk-ai/spekk-skills`) and the env var overrides for self-hosted mirrors
- The README section explains that `--force` is required to overwrite an existing skill
