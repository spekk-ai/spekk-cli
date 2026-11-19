---
slug: cli-reference-recipe-intro-says-export-token
type: code_spec_misalignment
severity: medium
status: open
created: 2026-11-19T00:00:00Z
affected:
  - docs/cli-reference.md
  - specs/sandbox-auth-mode/assertions/auth-modes-are-documented.md
---

# CLI reference setup recipes still open with "Export the variables the mode needs"

## Issue Description

The setup recipes in `docs/cli-reference.md` start with one instruction:
export every variable the mode needs, then run one command. For
`--auth subscription`, one of those variables is `CLAUDE_CODE_OAUTH_TOKEN`.
The recipe below that line, and the configuration reference it links to,
say never to export that token. Commits 910af1d and ec031c3 (#354) fixed
the recipe code blocks and the `--provider none` sentence. They left the
intro unchanged, so the page still contradicts itself.

## Evidence

- `docs/cli-reference.md` line 805, the intro to **Setup recipes**: "Each
  recipe pairs a provider with an auth mode. Export the variables the mode
  needs first ([Sandbox provisioning](configuration.md#sandbox-provisioning)
  lists them all), then run one command."
- `docs/configuration.md` line 212: the provisioning table that the intro
  links to lists `CLAUDE_CODE_OAUTH_TOKEN` as a variable the `subscription`
  mode needs.
- `docs/cli-reference.md` line 824: "The Claude token is set on the `spekk`
  command, not exported: an exported `CLAUDE_CODE_OAUTH_TOKEN` replaces your
  `/login` session in every `claude` process on the machine."
- `docs/configuration.md` line 266: "**Do not export the token from your
  shell profile.**"
- `specs/sandbox-auth-mode/assertions/auth-modes-are-documented.md`
  (`status: done`): "The configuration reference and the subscription setup
  recipe in `docs/cli-reference.md` give the token to the one `spekk` command
  that needs it, as an assignment in front of that command, not as an
  `export`."

## Impact

Operators read the intro first. The intro sends them to the table and tells
them to export everything in it. An operator who does that for the
subscription mode exports the token into their shell. Every interactive
`claude` session on their machine then uses the token instead of their
`/login` account, which is the failure the last three commits on this
assertion were meant to prevent. The code blocks are correct, but the
sentence above them says the opposite.

## Recommendation

Reword the intro so the Claude token is the exception. For example: "Export
the variables the mode needs first, except the Claude subscription token,
which goes in front of the one command (see below), then run one command."
Another option is to drop the blanket "export" and let each recipe show how
it sets its variables. The remedy commit sets the assertion back to `failed`
so the builder picks this up.
