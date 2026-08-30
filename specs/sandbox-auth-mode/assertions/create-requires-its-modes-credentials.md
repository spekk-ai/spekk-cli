---
id: create-requires-its-modes-credentials
parent: sandbox-auth-mode
created: 2026-08-27T00:00:00Z
priority: 1
status: done
branch: feat/sandbox-subscription-auth
depends-on: auth-flag-selects-mode
---

# The Pre-Flight Check Asks for the Credentials the Chosen Mode Uses

`Create` refuses to start when a variable it needs is missing. The list it checks depends on the auth mode, so a subscription sandbox is not blocked by absent AWS credentials it will never use, and a Bedrock sandbox is still blocked by them.

**Tests:** internal/sandbox/auth_test.go

## Success Criteria

- The required-variable list in `Create` (`internal/sandbox/commands.go`) is built by a helper, `requiredEnvVars(AuthMode) []string`, rather than a literal slice in the function body.
- `GITHUB_TOKEN` and `SPEKK_HOST` are required in both modes — the agent needs a git identity and a control host whatever it pays with.
- `bedrock` adds `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and `AWS_DEFAULT_REGION`, matching the list `Create` checks before this change.
- `subscription` adds `CLAUDE_CODE_OAUTH_TOKEN` and none of the AWS variables.
- The existing error text and its behavior are unchanged: one error naming every missing variable at once, comma-separated, so an operator fixes them in one pass rather than one run per variable.

## Verification

- `go test ./internal/sandbox` asserts the returned list per mode, including that `subscription` contains no string beginning `AWS_`.
- Running `spekk sandbox create --name probe --auth subscription` with `GITHUB_TOKEN`, `SPEKK_HOST`, and `CLAUDE_CODE_OAUTH_TOKEN` set and no AWS variables gets past the check and fails later, on the DigitalOcean call — not on a missing-variable error.
