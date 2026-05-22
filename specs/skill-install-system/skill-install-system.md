---
id: skill-install-system
created: 2026-05-22T12:00:00Z
priority: 1
---

# Skill Install System

## Overview

`spekk install` fetches a skill markdown file from an official GitHub registry (or any user-supplied URL) and writes it into one of the existing layered skill directories so the `SkillResolver` picks it up automatically. Companion commands (`spekk uninstall`, `spekk skills list`) round out the lifecycle.

Today, skills are either shipped inside the binary (embedded) or hand-authored by users under `.spekk/skills/<agent>/` or `~/.spekk/skills/<agent>/`. There is no way to grab a community-maintained skill without copying the file manually. This spec adds first-class install/uninstall/list commands that target the same directories the resolver already reads from — so nothing about resolution needs to change.

## Command Surface

```bash
# Install from official registry (default: local scope)
spekk install <agent> <skill>

# Install globally for the current user
spekk install <agent> <skill> --global

# Install from any downloadable markdown file
spekk install <agent> <skill> --source https://example.com/foo.md

# Overwrite an existing file
spekk install <agent> <skill> --force

# Remove an installed skill
spekk uninstall <agent> <skill> [--global|--local]

# List skills available to an agent across all scopes (local + global + embedded)
spekk skills list <agent>

# List skills available in the remote registry that aren't installed yet
spekk install --list <agent>
```

`<agent>` is one of `coach`, `builder`, `observer`. Any other value is rejected with an error listing the valid options.

## Scope Resolution

| Flag | Destination |
|------|-------------|
| *(none)* / `--local` | `<cwd>/.spekk/skills/<agent>/<skill>.md` |
| `--global` | `~/.spekk/skills/<agent>/<skill>.md` |

Local is the default because it matches `npm install` and the project-scoped precedence the resolver already gives to local files. If both `--local` and `--global` are passed, the command fails with an error (the user is asking for two things at once).

## Sources

### Official registry (default)

The official skill registry lives at **`github.com/spekk-ai/spekk-skills`** with a flat directory per agent:

```
spekk-skills/
├── coach/
│   ├── meeting-notes.md
│   └── business-model-validator.md
├── builder/
│   └── ...
└── observer/
    └── ...
```

- Raw fetch: `https://raw.githubusercontent.com/spekk-ai/spekk-skills/main/<agent>/<skill>.md`
- Listing (`--list`): `https://api.github.com/repos/spekk-ai/spekk-skills/contents/<agent>` (unauthenticated, 60 req/hr/IP)

Both base URLs can be overridden via env vars `SPEKK_SKILLS_RAW_BASE` and `SPEKK_SKILLS_API_BASE` so users can point at a fork or self-hosted mirror (and so tests can substitute an `httptest` server).

### Arbitrary `--source` URL

When `--source <URL>` is passed:

- The URL must be `http://` or `https://`; anything else is rejected.
- The content is written verbatim to the target path. No transformation, no validation that it's "really" markdown.
- The destination filename is the `<skill>` positional arg. If the user omits `<skill>`, fall back to the URL's basename minus `.md`; if that yields nothing usable (e.g. URL ends in `/`), fail with a clear error asking for an explicit name.

## Conflict Handling

If the destination file already exists at the chosen scope, the install refuses with a message that names the path and tells the user to pass `--force`. This protects local customizations and keeps scripts/CI deterministic. `--force` overwrites unconditionally.

## Uninstall

`spekk uninstall <agent> <skill>` deletes `<scope>/.spekk/skills/<agent>/<skill>.md`. Scope flags work the same as install. If the file doesn't exist, the command exits non-zero with a "not installed" error so scripts can detect it.

Uninstall never touches embedded skills or files outside the two on-disk scopes.

## Listing

- **`spekk skills list <agent>`** — wraps the existing `SkillResolver.ListSkills(agent)` so users can see everything currently available (local, global, embedded), with each entry's source dir. This is purely a display wrapper around code that already exists.
- **`spekk install --list <agent>`** — hits the registry contents API and prints the available remote skills, marking which are already installed locally or globally. Useful for discovery before committing to an install.

## Non-Goals

- **No versioning.** A skill is always the file at `main` (or whatever ref the env var override points at). Adding semver / locking is out of scope; users who need pinning can use `--source` with a commit-pinned raw URL.
- **No dependency graph between skills.** Each skill is a self-contained markdown file.
- **No mutation of embedded skills.** Install never touches the binary's embedded FS.
- **No npm wrapper.** Decided against `npx spekk-install` — the skills are useless without the Go binary, so requiring the binary as the entry point is the honest design.

## Assertions

See `assertions/` for what must be true about install, uninstall, and listing.
