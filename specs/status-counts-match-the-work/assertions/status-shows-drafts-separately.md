---
id: status-shows-drafts-separately
parent: status-counts-match-the-work
created: 2026-10-17T00:00:00Z
priority: 2
status: done
---

# `spekk status` shows draft assertions separately from the count

## Success Criteria

- The spec line counts only the assertions that are not draft: `✅ Access grants (3/3 assertions complete · 1 draft)`.
- A spec with no draft assertions prints as before: `(1/2 assertions complete)`.
- In "Overall Progress", `Total` and `Completion` leave out draft assertions, and a `📝 Draft: N` line shows when N is more than 0.
- Live check: `spekk status` on a spec with 3 done and 1 draft assertion shows `3/3` and `1 draft`.
