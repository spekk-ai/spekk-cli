---
id: announce-marks-branch
parent: observer-announce
created: 2026-07-26T12:00:00Z
priority: 1
status: not_started
depends-on: announce-conversation-open
---

# On Successful Delivery, Announce Commits the `announced:` Flip to the Observer Branch and Pushes

## Description

Success is recorded declaratively where the lifecycle spec says it lives: a
frontmatter commit on the observation's own branch, pushed to origin. This is
the Go-side half of the idempotent-retry contract.

## Success Criteria

- After the conversation open succeeds, announce commits exactly one change to
  the `observer/<slug>` branch: setting `announced: <ISO 8601 timestamp>` in
  the observation's frontmatter. No other files change in that commit.
- The commit is pushed to origin, so any other clone (sandbox rebuild, fresh
  checkout) sees the observation as announced after a fetch.
- The flip is written **only** on delivery success. Any failure — spool env
  unset, request-write error, selection error, push rejection — leaves the
  frontmatter without `announced:`, preserving eligibility for the next run
  (`announced-marker-retry` in the lifecycle spec).
- If the commit or push of the flip fails *after* a successful delivery, the
  command exits non-zero and logs the failure (see `announce-failure-log`);
  the documented worst case is one duplicate announcement on the next run —
  never a finding silently marked announced that no human ever saw.
- The commit message identifies the observation slug and the action (e.g.
  `observer: mark <slug> announced`), so branch history reads as a lifecycle
  log.

**Note:** ordering is deliberate — deliver first, then mark. The failure
domain is chosen so that the recoverable error (duplicate ping) is preferred
over the unrecoverable one (silent drop).
