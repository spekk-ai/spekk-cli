---
id: update-ci-for-go
parent: golang-cleanup
created: 2026-04-05T12:33:00Z
priority: 3
status: done
depends-on: remove-node-source
branch: feature/golang-cleanup
---

# CI/CD updated for Go build and test

GitHub Actions workflows updated to build and test the Go binary instead of Node.js.

## Success Criteria

- Test workflow runs `go test ./...` instead of `npm test`
- Build step runs `go build ./cmd/spekk`
- Publish workflow builds Go binaries for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
- Release creates GitHub release with platform-specific binaries
- No npm publish step remains
- Go version pinned in workflow (matches go.mod)
