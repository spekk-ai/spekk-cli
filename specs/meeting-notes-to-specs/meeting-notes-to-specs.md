---
id: meeting-notes-to-specs
created: 2025-02-12T19:30:00Z
priority: 2
status: not_started
---

# Meeting Notes to Specs

A coach skill that processes meeting transcripts and extracts actionable outcomes: todos, specs, and context updates.

## Overview

Meeting-to-spec conversion is a **coach responsibility**. When the user provides a meeting transcript, the coach activates its meeting-processing skill to categorize content into three outputs:

1. **Todos** - action items and follow-ups
2. **Specs** - features and product changes
3. **Context** - architectural decisions and patterns

This is not a standalone agent. It extends the coach's existing capabilities through the coach skills framework (see `specs/coach-skills-system/`). The coach prompt is the base — meeting processing is an additive skill/mode, not a separate prompt.

## Success Criteria

- Coach skill activates when user provides a meeting transcript via `spekk coach meeting`
- Meeting transcript → structured outputs in <30 seconds
- Todos separated from specs (action items ≠ product features)
- All outputs formatted correctly and ready to commit
- Single commit includes all three categories with clear labeling
- No duplication of spec format rules — reuses coach's existing knowledge of spec structure

## Workflow

```
Input: meeting transcript file (via `spekk coach meeting`)
↓
Coach activates meeting-processing skill
↓
Output 1: TODOS.md (action tracking)
Output 2: Proposed specs (waits for approval)
Output 3: Updated CONTEXT.md (shows diff for approval)
↓
Single commit with all changes
```
