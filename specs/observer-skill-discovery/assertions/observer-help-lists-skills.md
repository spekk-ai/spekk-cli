---
id: observer-help-lists-skills
parent: observer-skill-discovery
created: 2026-05-22T12:00:00Z
priority: 2
status: not_started
depends-on: skill-resolver-includes-observer
branch: feature/observer-skill-discovery
---

# Observer Help Dynamically Lists Available Skills

## Description

`spekk observer --help` lists available observer skills from all layers (local, global, package, embedded) by delegating to the shared `ShowHelp` helper in `internal/agent/launcher.go`, the same pattern coach and builder use.

## Success Criteria

- `spekk observer --help` (and `-h`, `help`) routes to `agent.ShowHelp(installDir, "observer")` instead of the hardcoded `showObserverHelp()` function
- Help output includes an `AVAILABLE SKILLS:` section listing observer skills found across all layers
- Help output continues to document observer-specific options (`--interval`, `--quiet`)
- Skills shadowed by a higher-priority layer appear only once in the listing
- When no skills are available, the help shows `(none found)` under the skills section (current `ShowHelp` behavior)
- The hardcoded `showObserverHelp()` function is removed or replaced — observer help comes from one source
