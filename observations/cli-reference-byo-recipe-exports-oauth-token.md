---
slug: cli-reference-byo-recipe-exports-oauth-token
type: code_spec_misalignment
severity: medium
status: open
created: 2026-11-13T00:00:00Z
affected:
  - docs/cli-reference.md
  - specs/sandbox-auth-mode/assertions/auth-modes-are-documented.md
---

# CLI reference still says to export the subscription token for a `--provider none` machine

## Issue Description

Commit 910af1d stopped the docs from telling operators to export
`CLAUDE_CODE_OAUTH_TOKEN`, and #353 added that rule to the
`auth-modes-are-documented` assertion. The commit fixed the DigitalOcean
subscription recipe in `docs/cli-reference.md` but not the `--provider none`
recipe a few lines further down. That recipe still says to "export
`CLAUDE_CODE_OAUTH_TOKEN` in place of the AWS keys".

## Evidence

- `docs/cli-reference.md` line 836 (the paragraph after the
  "A machine you already have (`--provider none`), Bedrock" block):
  "For a subscription machine, swap in `--auth subscription` and export
  `CLAUDE_CODE_OAUTH_TOKEN` in place of the AWS keys."
- The same file at line 824, from 910af1d: "The Claude token is set on the
  `spekk` command, not exported: an exported `CLAUDE_CODE_OAUTH_TOKEN`
  replaces your `/login` session in every `claude` process on the machine."
- `specs/sandbox-auth-mode/assertions/auth-modes-are-documented.md`
  (`status: done`): "The configuration reference and the subscription setup
  recipe in `docs/cli-reference.md` give the token to the one `spekk` command
  that needs it, as an assignment in front of that command, not as an
  `export`."
- `git grep 'export CLAUDE_CODE_OAUTH_TOKEN'` on main finds nothing, so the
  word "export" in prose is the one place left. A grep for the literal
  command does not find it.

## Impact

An operator who sets up a machine they already have, with a subscription,
follows the instruction the docs warn against. Every interactive `claude`
session on their machine then runs on the token instead of their `/login`
account. It reports API-key authentication and loses the features that need
the signed-in account. Two paragraphs of the same page also contradict each
other.

## Recommendation

Rewrite the sentence so the token goes in front of the one command. For
example: "swap in `--auth subscription` and replace the AWS keys with
`CLAUDE_CODE_OAUTH_TOKEN="$(cat ~/.config/spekk/oauth-token)"` in front of
the `spekk sandbox create` command, as in the subscription recipe above". The
remedy commit sets the assertion back to `failed` so the builder picks this
up.
