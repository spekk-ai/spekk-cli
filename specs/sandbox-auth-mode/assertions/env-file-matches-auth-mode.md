---
id: env-file-matches-auth-mode
parent: sandbox-auth-mode
created: 2026-08-27T00:00:00Z
priority: 1
status: done
branch: feat/sandbox-subscription-auth
depends-on: auth-flag-selects-mode
---

# The Injected Env File Carries One Mode's Credential and None of the Other's

`buildEnvContent` writes `/etc/spekk/agent.env` from the chosen auth mode. Each mode contributes its own credential block. The variables belonging to the mode that was not chosen are absent from the file, not merely overridden.

**Tests:** internal/sandbox/auth_test.go

## Success Criteria

- `buildEnvContent` takes the auth mode as a parameter and keeps its existing signature otherwise.
- Both modes write the shared lines exactly as today: `GITHUB_TOKEN`, `SPEKK_HOST` (still stripped of its scheme and trailing slash), `SPEKK_AGENT_TOKEN`, `WORKSPACE`, and `SPEKK_AGENT_NAME`.
- `bedrock` mode additionally writes `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_DEFAULT_REGION`, and `CLAUDE_CODE_USE_BEDROCK=1` — the same set, with the same values, that the function writes before this change.
- `subscription` mode additionally writes `CLAUDE_CODE_OAUTH_TOKEN` and nothing else.
- In `subscription` mode the rendered file contains none of the strings `CLAUDE_CODE_USE_BEDROCK`, `ANTHROPIC_API_KEY`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, or `AWS_DEFAULT_REGION`.
- In `bedrock` mode the rendered file contains neither `CLAUDE_CODE_OAUTH_TOKEN` nor `ANTHROPIC_API_KEY`.
- `injectCredentials` reads `CLAUDE_CODE_OAUTH_TOKEN` from the process environment alongside the variables it already collects, and passes the mode through to `buildEnvContent`.
- The file still ends with a single trailing newline, and `buildInjectScript` still base64-encodes it, so a credential containing a shell metacharacter cannot break out of the injection command.

**Note:** absence is the assertion. A test that only checks the token is present would pass against an implementation that appends the token to the Bedrock block and leaves the sandbox billing through Bedrock — the exact failure this spec exists to prevent.

## Verification

- One test pins the whole `bedrock` file, so any edit to the default path — a dropped shared line, or a leaked subscription token — fails loudly.
- A second test covers the other direction: `subscription` carries its token, the shared lines, and none of the credentials the pinned file carries. Two tests, one job each. A two-mode table was written first and cut: its `bedrock` row only re-asserted what the pin already proves.
- Both were checked by mutation. Making `subscription` emit `CLAUDE_CODE_USE_BEDROCK` fails the suite; reverting it passes. A test for absence that has never been seen to fail is not evidence of anything.
