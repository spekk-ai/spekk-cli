---
id: spec-not-started-until-work-starts
parent: status-counts-match-the-work
created: 2026-10-17T00:00:00Z
priority: 2
status: done
---

# A spec is `not_started` until one of its assertions starts

## Success Criteria

- All counted (not draft) assertions `not_started`: the spec is `not_started`.
- At least one counted assertion `in_progress` or `done`, and not all `done`: the spec is `in_progress`.
- All counted assertions `done`: the spec is `done`.
- Any counted assertion `failed`: the spec is `failed`, as before.
- No assertions, or only draft assertions: the spec is `not_started`, as before.
- The cross-branch explorer uses the same rule, through `ParentStatusFromChildStatuses`.
- Live check: `spekk status` on a spec with three `not_started` assertions shows 📋, not 🚧.
