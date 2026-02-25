---
id: reuses-coach-prompt-as-base
parent: meeting-notes-to-specs
created: 2026-02-24T23:59:00Z
priority: 1
status: done
---

# Reuses Coach Prompt as Base

**Tests:** src/coach/__tests__/reuses-coach-prompt-as-base.test.js

Meeting processing extends the coach prompt rather than defining a standalone agent prompt with duplicated rules.

## Success Criteria

- No standalone meeting-processor agent prompt exists
- The coach prompt is the base for meeting processing
- Meeting-processing skill adds behavior on top of the coach's existing capabilities
- Spec format definitions (YAML frontmatter, kebab-case IDs, priority levels, status values) are NOT duplicated — the coach already knows these
- Meeting-specific instructions (transcript categorization, todo extraction, context updates) are loaded as an additive skill
- The skill follows the extensible skills framework pattern from `specs/coach-skills-system/`
