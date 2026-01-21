---
id: asks-clarifying-questions
parent: coach-agent
created: 2026-01-20T17:10:00Z
priority: 3
status: done
---

# Coach Must Ask Clarifying Questions

## What Must Be True

When user requests are vague or incomplete, the coach must guide them through refinement by asking targeted questions.

## Question Categories

### 1. Scope Questions
- What exactly should happen?
- Which parts of the system are affected?
- Are there related features this depends on?

### 2. Testability Questions
- How will we know this is working?
- What does success look like?
- What are the edge cases?

### 3. Priority Questions
- How important is this? (1=critical, 2=important, 3=nice-to-have)
- Does this block other work?
- When does this need to be done?

### 4. Granularity Questions
- Should this be broken into smaller pieces?
- Are there multiple independent assertions here?
- Can parts be implemented separately?

## Example

**Vague request:** "Make the app faster"

**Coach questions:**
1. Which part feels slow? (page load, interactions, data fetching)
2. What's the current behavior? What should it be?
3. How will we measure improvement? (load time, response time)
4. What's the acceptable threshold? (< 2 seconds, < 500ms)

**Result:** Specific, measurable assertions about performance

## Success Criteria

- ✅ Coach identifies when requests are too vague
- ✅ Coach asks targeted questions to refine requirements
- ✅ Coach guides users toward testable assertions
- ✅ Coach helps determine appropriate priority
- ✅ Questions lead to well-formed specs

**Tests:** app/coach/__tests__/asks-clarifying-questions.test.js
