---
id: git-credentials-use-gh-helper
parent: sandbox-auth-mode
created: 2026-10-24T15:28:13Z
priority: 1
status: done
branch: fix/no-git-credential-store
---

# Git Uses the Existing GitHub Login Without a Credential Store

The terminal sandbox create and provision paths, and the credential setup script, already sign the agent in to GitHub CLI. Git must read that login when it needs a GitHub credential and must not keep a second plaintext token in the agent's Git credential file.

**Tests:** internal/sandbox/git_credentials_test.go

## Success Criteria

- Both credential setup paths configure Git to use `gh auth git-credential` for GitHub after the GitHub login succeeds. Clone, fetch, and push can request the credential without `credential.helper store` or a token in the Git remote URL.
- Neither path writes `/home/agent/.git-credentials`. Each path removes a stale file only after Git returns a GitHub credential through the new helper as `agent`. If the lookup fails, the path reports one `spekk:` error, exits with a failure, and keeps the stale file.
- A regression test runs both setup paths with real Git in fresh, legacy store, existing GitHub CLI URL helper, and machine-mode URL helper config states. It fails if Git cannot get the new credential, if the stale file is removed before lookup, or if either path writes the Git credential file.
- The machine-mode helper that asks the control host for installation tokens keeps its current behavior.

## Verification

- Run the sandbox package tests, the shell parser and shellcheck on the setup script, and `spekk validate`.
- Exercise Git credential lookup with a temporary home, a `su` shim, and a fake GitHub CLI. Check all four config states and the stale file removal order.
