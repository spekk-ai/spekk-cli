---
id: announce-conversation-open
parent: observer-announce
created: 2026-07-26T12:00:00Z
priority: 1
status: not_started
depends-on: announce-selection
---

# The Announcement Is Delivered via `spekk conversation open` With a Fixed Message Shape

## Description

Delivery reuses the existing sandbox conversation mechanism — announce writes
one conversation-open request for the selected observation, with a title and
body whose shape is fixed in code.

## Success Criteria

- For the selected observation, announce performs the equivalent of
  `spekk conversation open` (same spool contract as
  `specs/sandbox-conversation-open/`: one request into the directory named by
  `SPEKK_CONVERSATION_SPOOL`) with:
  - **title** = the finding's title
  - **body** containing, in order: a 2–3 sentence summary of the evidence
    (drawn from the observation body and `affected` paths); the pointer
    `Proposed fix in PR: <url> — merge to accept, close to dismiss. Reply
    here to discuss.`; and a severity warning reflecting the observation's
    severity
- The message shape lives in Go (template/format in code), not in a prompt —
  the observer prompt contains no instructions for composing announcement
  text.
- **Sandbox constraint, documented and enforced:** when
  `SPEKK_CONVERSATION_SPOOL` is unset or empty, the command exits non-zero
  with a message stating it must run inside a sandbox session; it does not
  claim success and does not write the `announced:` flip. The constraint is
  documented in the parent spec and in the command's `--help` text.
- The `close to dismiss` wording is consistent with the lifecycle convention
  (`pr-body-conventions`): closing parks; deletion forgets.

**Note:** if `pr:` is absent from the frontmatter at announce time, follow
the resolution of the parent spec's open question (proposed: substitute the
branch reference for the PR URL rather than blocking the announcement) —
whichever way it is settled, record the decision here.
