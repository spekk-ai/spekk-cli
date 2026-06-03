---
id: gemfury-publish
parent: release-flow
created: 2026-06-03T18:00:00Z
priority: 1
status: done
depends-on: cross-compile-binaries
branch: temporary-target
---

# Release script uploads versioned artifacts to Gemfury

A release script packages and pushes the cross-compiled binaries to Gemfury so that only users with a valid Gemfury token can download them.

## Success Criteria

- Release script uploads all 6 binaries to a Gemfury private repository
- Artifacts are versioned (e.g., `spekk-darwin-arm64-v1.2.3`)
- Upload requires a `GEMFURY_TOKEN` environment variable
- Script fails clearly if token is missing or upload fails
- Published artifacts are downloadable via authenticated Gemfury URL (e.g., `https://TOKEN@fury.io/ACCOUNT/spekk-darwin-arm64-v1.2.3`)
- Old versions remain available (no overwrite unless explicit)
