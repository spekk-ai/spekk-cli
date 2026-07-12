---
id: qa-docs-site
parent: docs-three-sites
created: 2026-06-04T14:00:00Z
priority: 2
status: not_started
branch: feature/docs-three-sites
---

# QA docs site exists at `docs-qa/`

A Zensical site in `docs-qa/` provides structured release testing checklists and manual verification procedures.

## Success Criteria

- `docs-qa/` directory exists with its own Zensical config
- **Release checklist page** (`docs-qa/release-checklist.md`):
  - Pre-release verification steps (tests pass, version embedding works, binaries compile)
  - Platform-specific checks for each of the 6 build targets
  - Self-update flow verification (`spekk update --check`, then full update)
  - Smoke test commands (`spekk --version`, `spekk show`, `spekk next`)
- **Manual verification page** (`docs-qa/manual-verification.md`):
  - Agent launch verification (coach, builder, observer start without errors)
  - `spekk serve` WebSocket connectivity check
  - `spekk show --watch` live reload verification
  - Sandbox create/destroy lifecycle (if DO credentials available)
- Navigation is defined in the Zensical config
