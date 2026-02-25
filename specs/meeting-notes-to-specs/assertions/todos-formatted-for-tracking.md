---
id: todos-formatted-for-tracking
parent: meeting-notes-to-specs
created: 2025-02-12T19:30:00Z
priority: 2
status: done
---

# Todos Formatted for Action Tracking

**Tests:** src/coach/__tests__/todos-formatted-for-tracking.test.js

Todo items extracted from meetings by the coach's meeting-processing skill are formatted and output to TODOS.md for tracking.

## Success Criteria

- Todo items include: description, owner (if mentioned in meeting), context
- Outputs to `TODOS.md` in structured markdown format
- Format: `- [ ] {description} (@{owner}) - from meeting {date}`
- Example: `- [ ] Send spreadsheet to team (@marcy) - from meeting 2025-02-12`
- Non-spec action items don't become specs (e.g., "follow up with Kaiser" is a todo, not a spec)
- Todos are actionable and clearly separated from product features
