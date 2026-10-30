---
id: machine-model-credential
parent: sandbox-machine-mode
created: 2026-10-17T00:00:00Z
priority: 1
status: done
branch: feature/sandbox-machine-mode
depends-on: machine-registers-with-one-time-token
---

# The Env File Carries the Model Credential of the Chosen Kind, and Nothing of the Other Kinds

As in `specs/sandbox-auth-mode/`, absence is the requirement: a variable of another kind in the file can select a credential path that the caller did not choose.

**Tests:** internal/sandbox/auth_test.go

## Success Criteria

- `bedrock` writes `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_DEFAULT_REGION` (from `aws_region`), and `CLAUDE_CODE_USE_BEDROCK=1`, as the terminal `bedrock` mode does, plus the model pin `ANTHROPIC_MODEL=us.anthropic.claude-opus-5-5`.
- `anthropic_api_key` writes `ANTHROPIC_API_KEY` and the model pin `ANTHROPIC_MODEL=claude-opus-5-5`, and no other model line.
- `none` writes no model credential line and no model pin.
- The model pin is not a credential. Its values follow `specs/sandbox-auth-mode/assertions/env-file-matches-auth-mode.md`, and the machine input has no field that changes it.
- In each kind, the file has none of `CLAUDE_CODE_OAUTH_TOKEN`, `ANTHROPIC_API_KEY`, `CLAUDE_CODE_USE_BEDROCK`, or an AWS variable, except the lines that the kind itself writes.
- The shared lines in machine mode: `SPEKK_HOST`, `SPEKK_AGENT_TOKEN`, `SPEKK_AGENT_ID`, `WORKSPACE`, and `SPEKK_AGENT_NAME`. There is no `GITHUB_TOKEN` line: the installation token lives 1 hour, and git gets a new one from the helper of `machine-git-access`.
- The record stores the kind as `auth` (`none`, `api-key`, or `bedrock`), and no credential.
- The env file goes to the machine on the stdin of `ssh` (`secrets-on-ssh-stdin`).

## Verification

- One test renders the file for `none` and for `anthropic_api_key`, and checks the lines of the type and the absence of each other credential variable. The pinned `bedrock` test of `sandbox-auth-mode` already proves the Bedrock block.
