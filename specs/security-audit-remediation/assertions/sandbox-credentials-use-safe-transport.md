---
id: sandbox-credentials-use-safe-transport
parent: security-audit-remediation
created: 2026-06-03T12:00:00Z
priority: 1
status: done
branch: feature/spekk-sandbox-vulnrabilities
---

# Sandbox credential injection uses safe transport instead of heredoc interpolation

The `injectCredentials` and `configureGitCredentials` functions in `internal/sandbox/commands.go` do not interpolate environment variable values (AWS keys, GITHUB_TOKEN, SPEKK_AGENT_TOKEN) into shell command strings. A credential value containing newlines, quotes, or heredoc terminators cannot break out of the intended context and execute arbitrary commands on the sandbox.

## Success Criteria

- `injectCredentials` does not embed credential values via `fmt.Sprintf` into a shell heredoc — instead uses a safe mechanism (e.g., base64-encode the env file content in Go, decode on the remote side; or SCP a temp file)
- `configureGitCredentials` does not interpolate `GITHUB_TOKEN` into shell command strings via `fmt.Sprintf` — instead pipes the token via stdin or uses a file-based approach
- A GITHUB_TOKEN value containing `"; rm -rf /; echo "` does not execute shell commands
- An AWS_SECRET_ACCESS_KEY containing `\nENVEOF\nmalicious_command` does not break out of the heredoc
- Existing sandbox create/destroy workflows still function correctly after the change
- Tests cover injection attempts for both functions

**Tests:** internal/sandbox/commands_test.go
