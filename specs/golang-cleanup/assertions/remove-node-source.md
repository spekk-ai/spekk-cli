---
id: remove-node-source
parent: golang-cleanup
created: 2026-04-05T12:32:00Z
priority: 3
status: not_started
depends-on: node-parser-removed
branch: feature/golang-migration
---

# All Node.js source code removed

After the Go binary handles all commands, all Node.js implementation code is deleted.

## Success Criteria

- `src/` directory deleted entirely
- `bin/spekk.js` deleted
- `package.json`, `package-lock.json` deleted
- `node_modules/` deleted (and in .gitignore)
- No `.js` files remain in the project root or source directories
- Go binary is the sole `spekk` executable
- `go build ./cmd/spekk` is the only build step required
