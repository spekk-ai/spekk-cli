---
id: announce-conversation-open
parent: observer-announce
created: 2026-07-26T12:00:00Z
priority: 1
status: done
depends-on: announce-selection
---

# The Announcement Is Delivered via `spekk conversation open` With a Fixed Message Shape

## Description

Delivery reuses the existing sandbox conversation mechanism — announce writes
one conversation-open request for the selected observation, with a title and
body whose shape is fixed in code.

## Success Criteria

- For the selected observations, announce performs the equivalent of
  `spekk conversation open` (same spool contract as
  `specs/sandbox-conversation-open/`: ONE request into the directory named by
  `SPEKK_CONVERSATION_SPOOL`), one message per run:
  - With a single finding: **title** = the finding's title; **body**
    containing, in order: a 2–3 sentence summary of the evidence (drawn from
    the observation body); the pointer `Proposed fix in PR: <url> — merge to
    accept, close to dismiss. Reply here to discuss.`; and a severity warning
    reflecting the observation's severity
  - With two or three findings: **title** = `Observer: N findings (X high,
    Y medium)`; **body** = one compact numbered section per finding (title
    with severity, the same 2–3 sentence summary, and the pointer `Proposed
    fix in PR: <url> — merge to accept, close to dismiss.`), then one shared
    footer: `Reply here to discuss.` plus the warning line of the highest
    severity present. Sections keep the selection order. Per-finding text
    stays at the single-finding size.
- The message body contains no list of `affected` paths.
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

**Decision (recorded):** when `pr:` is absent at announce time, the
announcement proceeds with the branch reference substituted into the same
pointer line: `Proposed fix in PR: observer/<slug> — merge to accept, close
to dismiss. Reply here to discuss.` The branch — not the PR — is the state
carrier, so a missing PR never blocks an announcement.

**Decision (recorded, 2026-07-27):** several same-run findings share one
message (cap three), instead of one message per finding.

**Decision (recorded, 2026-07-29):** the message shows no `Evidence:` line.
The first announcements in production put a full path list in the chat
message. The list is too long, it repeats the PR content, and it moves the
pointer line out of view. Evidence keeps its other two roles: an observation
with no `affected` path stays invalid (`announce-selection`), and the
observation file and the PR body still carry the paths.

**Tests:** internal/observer/announce_test.go
(TestAnnounceSuccessDeliversAndMarks, TestComposeRequestWithPRURL,
TestComposeBatchMultiple, TestAnnounceFailsLoudlyWithoutSpool)
