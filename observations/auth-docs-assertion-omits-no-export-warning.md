---
slug: auth-docs-assertion-omits-no-export-warning
type: outdated_specs
severity: low
status: resolved
created: 2026-11-12T00:00:00Z
affected:
  - specs/sandbox-auth-mode/assertions/auth-modes-are-documented.md
  - docs/configuration.md
  - docs/cli-reference.md
---

# Auth-docs assertion omits the warning against exporting the subscription token

## Issue Description

Commit 910af1d changed how the docs tell an operator to supply
`CLAUDE_CODE_OAUTH_TOKEN`. It is no longer exported from the shell. Instead it
is assigned in front of the single `spekk` command that needs it. The commit
also added a bold warning that says why: Claude Code prefers the variable over
the `/login` session, so exporting it moves every interactive `claude` session
on the machine onto the token. Those sessions then report API-key auth and lose
the features that need a signed-in account.

`auth-modes-are-documented` is the assertion that governs this documentation.
It says where the token is kept ("a file only the operator can read, never a
command line"), but it does not say how the token should reach the command, and
it does not require the warning. The docs now promise more than the spec does.

## Evidence

- `docs/configuration.md` (around lines 258–266): the recipe now reads
  `CLAUDE_CODE_OAUTH_TOKEN="$(cat ~/.config/spekk/oauth-token)" spekk sandbox create ...`,
  followed by "**Do not export the token from your shell profile.** ...".
- `docs/cli-reference.md` (around lines 816–824): the DigitalOcean subscription
  recipe makes the same change and adds a one-line pointer to that warning.
- `specs/sandbox-auth-mode/assertions/auth-modes-are-documented.md`, success
  criterion "The documentation says where to keep the token: …". Nothing in it
  covers export versus a per-command assignment, and nothing covers the warning.

## Impact

The warning describes a side effect that the operator cannot see: their own
interactive sessions quietly switch credentials. No spec records it, so a later
docs rewrite done against this assertion could bring back the `export` line
and still pass every criterion.

## Recommendation

Add a success criterion to `auth-modes-are-documented` that requires the docs
to pass the token on the one `spekk` command that needs it, and to warn against
exporting it, with the reason. The remedy commit on this branch proposes that
wording.
