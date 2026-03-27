---
id: meeting-notes-to-specs
created: 2026-01-23T22:14:00Z
priority: 2
---

# Meeting Notes to Specs

Processes meeting transcripts and converts feature discussions into proper spec files with assertions, todos, and context updates.

## Triggers

- "meeting notes"
- "meeting transcript"
- "meeting summary"
- "process meeting"
- "from our meeting"
- "discussed in meeting"
- "meeting action items"
- "meeting outcomes"
- "standup notes"
- "retro notes"
- "planning notes"
- "kickoff notes"

## Workflow

1. Receive meeting transcript or notes
2. Extract and categorize into three types:
   - Todos: Action items, follow-ups, assignments
   - Features: Product changes, new functionality requiring specs
   - Decisions: Architectural decisions, patterns, context updates
3. For each feature identified:
   - Create parent spec with title, description, priority
   - Break down into assertions (sub-tasks)
   - Assign priorities to assertions
   - Define success criteria
4. Format todos with owner and meeting date
5. Format decisions for CONTEXT.md with date stamp
6. Present proposal to user showing:
   - All specs to be created
   - Assertions for each spec
   - Priorities
   - Todos list
   - Context updates
7. Upon approval, create all files:
   - specs/{spec-id}/{spec-id}.md (parent specs)
   - specs/{spec-id}/assertions/{assertion-id}.md (assertions)
   - TODOS.md (updated with new todos)
   - CONTEXT.md (updated with decisions)
8. Stage and commit all files in single commit
9. Use commit message format: "Process meeting: {date} - {summary}"

## Validation

- Meeting transcript successfully parsed
- All content categorized (todos/features/decisions)
- Features converted to proper spec structure
- All spec files follow naming convention (kebab-case)
- Frontmatter includes: id, parent, created, priority, status
- Assertions properly linked to parent specs
- TODOS.md entries include meeting date reference
- CONTEXT.md entries include date stamp and context
- All files committed with descriptive message
- No uncategorized content remains

## Spec Structure

Parent spec frontmatter:
```yaml
---
id: feature-name
created: 2026-01-23T22:14:00Z
priority: 1-3
---
```

Assertion frontmatter:
```yaml
---
id: assertion-name
parent: feature-name
created: 2026-01-23T22:14:00Z
priority: 1-3
status: not_started
---
```

## Todo Format

```markdown
- [ ] Task description (@owner) - from meeting YYYY-MM-DD
```

## Context Decision Format

```markdown
- Decision from meeting YYYY-MM-DD: [decision text]
  - *Context: [why this decision was made]*
```

## Output Files

- `specs/{spec-id}/{spec-id}.md` - Parent spec file
- `specs/{spec-id}/assertions/{assertion-id}.md` - Assertion files
- `TODOS.md` - Updated todo list
- `CONTEXT.md` - Updated context with decisions

## Commit Message Structure

```
Process meeting: YYYY-MM-DD - Brief summary

Todos:
- Todo description 1
- Todo description 2

Specs created:
- spec-id-1
- spec-id-2

Context updates:
- Decision 1
- Decision 2
```
