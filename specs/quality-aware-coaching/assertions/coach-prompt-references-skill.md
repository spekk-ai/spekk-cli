---
id: coach-prompt-references-skill
parent: quality-aware-coaching
created: 2026-03-19T18:02:00Z
priority: 1
status: draft
depends-on: quality-skill-exists
---

# Coach prompt lists the quality-aware assertions skill

The coach agent prompt (`specs/coach-agent/coach.prompt.md`) references the quality-aware assertions skill in its Available Skills section so the coach knows to check for it.

## Success Criteria

- Coach prompt's "Available Skills" section (step 1.5) includes an entry for the quality-aware assertions skill
- Entry includes: skill name, trigger phrases, and brief description
- Trigger phrases include: "new API", "new endpoint", "build a feature", "add a page", or auto-detect when coach is writing assertions that involve backend/frontend work
- Coach prompt's skill detection workflow applies to this skill the same way it applies to existing skills (suggest → wait → apply if yes)
