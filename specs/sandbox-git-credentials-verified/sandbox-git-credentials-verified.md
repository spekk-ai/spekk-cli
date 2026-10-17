---
id: sandbox-git-credentials-verified
created: 2026-10-17T00:00:00Z
priority: 2
---

# A sandbox create fails when the GitHub credential does not land

## Problem

`buildGitCredentialScript` runs `gh auth login --with-token 2>/dev/null || true`. When the login fails, for example because the token is expired or not valid, the script hides the error and exits 0. `spekk sandbox create` and `spekk sandbox provision` then report success, and the agent on the sandbox cannot clone or push. Nobody sees the cause until a run fails (issue #48).

## Solution

The script does not hide a failure of `gh auth login`, and it runs `gh auth status` as the agent user after the login. A failure of either stops the script with a clear line on stderr. The command then fails at the "configuring git credentials" stage with that output and exits non-zero, as for the other stages.

## What this deliberately does not do

- **No `git ls-remote` check.** It needs a repository that the token can read, and the CLI does not know one. `gh auth status` checks that the token is valid with the GitHub API.
- **No check on a schedule.** That is a different feature.
