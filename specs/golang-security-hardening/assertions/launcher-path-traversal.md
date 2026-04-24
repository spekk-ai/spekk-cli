---
id: launcher-path-traversal
parent: golang-security-hardening
created: 2026-04-24T12:00:00Z
priority: 1
status: done
branch: feature/golang-migration
---

# Agent launcher validates file paths against traversal

The `BuildSkillMessage` function in `internal/agent/launcher.go` validates that resolved transcript file paths do not escape the current working directory via `../` traversal. Paths that resolve outside the working directory are rejected with an error.

## Success Criteria

- After resolving the transcript path, the function checks that the resolved absolute path has the working directory as a prefix
- Paths like `../../etc/passwd` are rejected with a clear error message
- Legitimate relative paths (e.g. `notes/meeting.txt`) still work
- Absolute paths within the working directory still work
- Test covers both accepted and rejected paths
