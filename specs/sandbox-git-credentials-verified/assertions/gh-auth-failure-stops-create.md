---
id: gh-auth-failure-stops-create
parent: sandbox-git-credentials-verified
created: 2026-10-17T00:00:00Z
priority: 2
status: not_started
---

# A failed `gh auth login` stops the credential stage with a clear error

## Success Criteria

- The git credential script has no `|| true` and no `2>/dev/null` on `gh auth login`.
- After the login, the script runs `gh auth status` as the agent user. When it fails, the script prints a line that says that the agent user is not logged in to GitHub, and exits non-zero.
- A failure of the script makes `configureGitCredentials` return an error, so `spekk sandbox create` and `spekk sandbox provision` exit non-zero.
- The output does not show the token.
- Live check: run the script in a local container with an `agent` user. A valid token exits 0, and `gh auth status` as `agent` exits 0. A token that is not valid exits non-zero with the error.
