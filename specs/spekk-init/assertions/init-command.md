---
id: init-command
parent: spekk-init
created: 2026-06-11T00:00:00Z
priority: 1
status: done
---

# `spekk init` Creates the specs/ Directory

## Success Criteria

- `spekk init` creates `specs/` at the git root when run inside a repository, otherwise in the current directory (same resolution as `findSpecsDir`)
- Writes `specs/README.md` explaining the spec/assertion layout and listing the core commands, so the directory is non-empty (git-trackable) and self-explanatory
- Idempotent: if `specs/` already exists it prints a friendly "already set up" message and exits 0 without touching anything
- On success, prints next steps: `spekk coach`, `spekk builder`, and the `spekk install --target` route for other assistants
- `spekk init --help` shows usage; `spekk help` lists `init` first in the command list
- The installed agent shims reference `spekk init` (not a vague pointer) when they encounter a project without `specs/`
- Documented in README Quick Start (step 1 of the workflow) and `docs/cli-reference.md`

**Tests:** verified via CLI smoke test (fresh dir, git repo subdir, and already-initialized cases).
