---
id: install-covers-hermes-and-gemini
parent: configurable-harness
created: 2026-09-06T00:00:00Z
priority: 1
status: in_progress
locked-by: builder-MacBook-Pro.local-63521-1788727954
depends-on: harness-profile-abstraction
branch: feature/configurable-harness
---

# `spekk install` covers hermes and gemini

`spekk install` supports claude-code, opencode, codex, copilot, and cursor, but
not hermes or gemini — so the auto-ensure step in
`interactive-uses-installed-skill` has nothing to call for those two. This adds
the missing coverage.

## Success Criteria

- `spekk install --target hermes` and `spekk install --target gemini` succeed and
  are listed by `spekk install`'s valid targets.
- Each writes the spekk agent instructions into that harness's native
  auto-loaded location, confirmed against the real harness: hermes into its skills
  directory (loadable via `hermes chat -s`), gemini into a `GEMINI.md` context
  file it auto-reads.
- Both global and project scope are handled the way the harness supports them; a
  scope the harness cannot support is a clear no-op, not a silent wrong write.
- Installs are idempotent and, for a shared context file like `GEMINI.md`, update
  only the spekk-owned section without clobbering other content.
- A test covers the hermes and gemini install destinations.
