---
id: extract-three-categories
parent: meeting-notes-to-specs
created: 2025-02-12T19:30:00Z
priority: 2
status: draft
---

# Extract Three Categories from Transcript

The coach's meeting-processing skill identifies and separates todos, features/specs, and decisions/context from meeting transcripts.

## Success Criteria

- Coach reads meeting transcript file (markdown, text, or copied transcript)
- Identifies and separates three distinct categories:
  - Todos (action items, follow-ups, assignments)
  - Features/specs (product changes, new functionality)
  - Decisions/context (architectural decisions, patterns established)
- Categorization is accurate: todos ≠ features ≠ decisions
- Each category extracted as separate output for further processing
