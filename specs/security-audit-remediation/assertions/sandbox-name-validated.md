---
id: sandbox-name-validated
parent: security-audit-remediation
created: 2026-06-03T12:00:00Z
priority: 1
status: not_started
branch: feature/spekk-sandbox-vulnrabilities
---

# Sandbox name is validated before use in shell commands

The user-supplied `--name` flag for sandbox commands is validated to contain only safe characters before being embedded in any shell command or environment variable. A malicious name cannot inject shell commands or break out of string contexts.

## Success Criteria

- Sandbox name is validated against `^[a-z0-9][a-z0-9-]*$` (lowercase alphanumeric and hyphens, must start with alphanumeric)
- Validation occurs at the CLI flag parsing layer, before the name reaches any shell command construction
- Names containing spaces, quotes, semicolons, newlines, backticks, or `$()` are rejected with a clear error message
- The validated name is used in `injectCredentials` (line 537) and any other location where name appears in shell context
- Tests cover accepted names (`my-sandbox`, `prod1`) and rejected names (`; rm -rf /`, `$(whoami)`, `name\nmalicious`)
