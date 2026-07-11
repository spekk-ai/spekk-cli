---
id: engineering-docs-site
parent: docs-three-sites
created: 2026-06-04T14:00:00Z
priority: 2
status: not_started
branch: feature/docs-three-sites
---

# Engineering docs site exists at `docs-engineering/`

A Zensical site in `docs-engineering/` with its own `zensical.toml` documents how spekk works internally — for people building and contributing to spekk.

## Success Criteria

- `docs-engineering/` directory exists with a `zensical-engineering.toml` at project root (or `zensical.toml` inside `docs-engineering/`)
- **Architecture page** (`docs-engineering/architecture.md`):
  - Internal package overview covering all 9 packages (`agent`, `cli`, `parser`, `sandbox`, `serve`, `show`, `status`, `update`, `version`)
  - Mermaid diagram showing how `cmd/spekk/main.go` routes commands to packages
  - Data flow from spec files → parser → JSON output
  - Agent launch sequence (coach, builder, observer)
- **Contributing page** (`docs-engineering/contributing.md`):
  - Content migrated from `CONTRIBUTING.md` (dev setup, testing, versioning, release workflow, project structure)
  - Expanded with internal conventions not in the root file
- **Observer internals page** (`docs-engineering/observer.md`):
  - Drift detection types and how they work
  - Observation file format (YAML frontmatter fields)
  - How observer output feeds back into coach workflow
- Navigation is defined in the Zensical config
