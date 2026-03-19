---
id: gate-loader-parses-gate-files
parent: reviewer-agent
created: 2026-03-19T18:08:00Z
priority: 1
status: not_started
branch: feature/code-quality-qa
---

# Gate loader parses .gate.md files from layered paths

A gate loader module (`src/reviewer/gate-loader.js`) reads `.gate.md` files from the three-layer resolution path (package → global → local) and parses their frontmatter and precondition sections.

## Success Criteria

- Module exists at `src/reviewer/gate-loader.js`
- Loads `.gate.md` files from three paths in order:
  1. Package gates: `<spekk-package-root>/gates/`
  2. Global gates: `~/.spekk/gates/`
  3. Local gates: `.spekk/gates/` (relative to cwd)
- Local gates override global gates override package gates (matched by `id` in frontmatter)
- Parses YAML frontmatter: `id`, `phase`, `tags`, `depends-on`
- Parses `## Preconditions` section into structured check objects:
  - `files-changed: "glob"` → `{ type: 'files-changed', pattern: 'glob' }`
  - `dir-exists: "path"` → `{ type: 'dir-exists', path: 'path' }`
  - `file-exists: "path"` → `{ type: 'file-exists', path: 'path' }`
  - `file-not-exists: "path"` → `{ type: 'file-not-exists', path: 'path' }`
  - `branch-matches: "pattern"` → `{ type: 'branch-matches', pattern: 'pattern' }`
  - `has-dependency: "pkg"` → `{ type: 'has-dependency', package: 'pkg' }`
  - `command-succeeds: "cmd"` → `{ type: 'command-succeeds', command: 'cmd' }`
- Parses `## LLM Judgment` section as raw text (passed to reviewer agent later)
- Parses `## Workflow` section as raw text (the actual check instructions)
- Parses `## On Failure` section for severity and action
- Returns array of gate objects sorted by dependency order
- Tests exist at `src/reviewer/__tests__/gate-loader.test.js`
