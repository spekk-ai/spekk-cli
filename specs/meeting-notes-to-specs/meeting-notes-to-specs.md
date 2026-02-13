---
id: meeting-notes-to-specs
created: 2025-02-12T19:30:00Z
priority: 2
status: not_started
---

# Meeting Notes to Specs

Process meeting transcripts and extract actionable outcomes: todos, specs, and context updates.

## Overview

The meeting-processor agent reads meeting transcripts and categorizes content into three outputs:
1. **Todos** - action items and follow-ups
2. **Specs** - features and product changes
3. **Context** - architectural decisions and patterns

## Success Criteria

- Meeting transcript → structured outputs in <30 seconds
- Todos separated from specs (action items ≠ product features)
- All outputs formatted correctly and ready to commit
- Single commit includes all three categories with clear labeling

## Workflow

```
Input: meeting transcript file
↓
Agent processes and categorizes
↓
Output 1: TODOS.md (action tracking)
Output 2: Proposed specs (waits for approval)
Output 3: Updated CONTEXT.md (shows diff for approval)
↓
Single commit with all changes
```
