---
id: decisions-update-context
parent: meeting-notes-to-specs
created: 2025-02-12T19:30:00Z
priority: 2
status: done
---

# Decisions Update CONTEXT.md

Architectural decisions and patterns discussed in meetings are extracted by the coach's meeting-processing skill and added to CONTEXT.md.

**Tests:** src/coach/__tests__/decisions-update-context.test.js

## Success Criteria

- Architectural decisions identified from meeting transcript
- Examples of decisions:
  - "Use deep-link searches instead of scraping"
  - "Partner with platforms rather than replacing them"
  - "Keep todos separate from specs in workflow"
- Appends to `CONTEXT.md` (creates file if it doesn't exist)
- Shows diff of proposed CONTEXT.md changes for user approval before updating
- Preserves existing CONTEXT.md structure and formatting
- Decisions formatted clearly with context from the meeting
- Date stamp included: "Decision from meeting 2025-02-12: ..."
