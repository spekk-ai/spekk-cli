---
id: git-credentials-use-gh-helper
parent: sandbox-auth-mode
created: 2026-10-24T15:28:13Z
priority: 1
status: in_progress
locked-by: builder-spekk-tmp-art-27905-1792855718
branch: fix/no-git-credential-store
---

# Git Uses the Existing GitHub Login Without a Credential Store

The terminal sandbox create and provision paths, and the credential setup script, already sign the agent in to GitHub CLI. Git must read that login when it needs a GitHub credential and must not keep a second plaintext token in the agent's Git credential file.

## Success Criteria

- Both credential setup paths configure Git to use `gh auth git-credential` for GitHub after the GitHub login succeeds. Clone, fetch, and push can request the credential without `credential.helper store` or a token in the Git remote URL.
- Neither path writes `/home/agent/.git-credentials`. Each path removes a stale file only after the GitHub login and the new helper are in place, so a failed setup does not remove the working credential of an existing agent.
- A regression test fails if either path restores `credential.helper store` or writes the Git credential file.
- The machine-mode helper that asks the control host for installation tokens keeps its current behavior.

## Verification

- Run the sandbox package tests, the shell parser and shellcheck on the setup script, and `spekk validate`.
- Exercise Git credential lookup with a temporary home and a fake GitHub CLI. Check that Git receives a credential for GitHub and does not create `.git-credentials`.
