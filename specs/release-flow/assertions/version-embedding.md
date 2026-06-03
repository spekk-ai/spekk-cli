---
id: version-embedding
parent: release-flow
created: 2026-06-03T18:00:00Z
priority: 1
status: done
branch: temporary-target
---

# Binary embeds a version string

The spekk binary has a version baked in at build time via `ldflags` and exposes it through `spekk --version`.

## Success Criteria

- A `version` variable exists in the main package (default: `"dev"`)
- `go build -ldflags "-X main.version=1.2.3"` sets the version at compile time
- `spekk --version` and `spekk version` print the version and exit
- Version string is accessible from other packages that need it (e.g., self-update, user-agent headers)
