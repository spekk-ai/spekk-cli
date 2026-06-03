---
id: transcript-handler-allows-any-user-path
parent: security-audit-remediation
created: 2026-06-03T12:00:00Z
priority: 1
status: in_progress
branch: feature/spekk-sandbox-vulnrabilities
---

# Transcript handler allows any user-specified file path

The `BuildSkillMessage` function in `internal/agent/launcher.go` does not restrict transcript file paths to the working directory. Since this is a local CLI tool where the user explicitly provides the path argument, they should be able to reference files anywhere on their filesystem (e.g., `~/Downloads/meeting.txt`, `/tmp/standup.txt`). The working directory restriction is removed. The function validates only that the resolved path is a regular file that exists.

## Success Criteria

- The working directory prefix check is removed — no path is rejected based on its location
- `spekk coach meeting ~/Downloads/notes.txt` works (absolute path outside working directory)
- `spekk coach meeting /tmp/standup.txt` works (absolute path in /tmp)
- `spekk coach meeting ../sibling-project/notes.txt` works (relative path outside working directory)
- `spekk coach meeting notes.txt` still works (relative path in working directory)
- Non-existent paths still return a clear "file not found" error
- The resolved path must be a regular file (not a directory, device, or symlink to one)
- Tests cover absolute paths, relative paths, tilde-expanded paths, non-existent files, and non-regular files
