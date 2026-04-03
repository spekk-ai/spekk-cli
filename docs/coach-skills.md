---
icon: lucide/graduation-cap
---

# Coach Skills

The coach has specialized skills for common development tasks. Skills are markdown files that the coach reads and follows as workflow instructions.

## How skills work

1. **Trigger detection** -- Coach detects keywords in your input
2. **Skill activation** -- Coach reads the skill markdown file
3. **Workflow execution** -- Coach follows the workflow steps
4. **Validation** -- Coach validates output against success criteria

Skills live in `specs/coach-skills-system/`.

---

## Meeting notes to specs

Process meeting transcripts into structured outputs.

**CLI:** `spekk coach meeting [file]`

**Triggers:** "meeting notes", "meeting transcript", "process meeting", "standup notes"

### What it extracts

=== "Todos"

    Appended to `TODOS.md`:

    ```markdown
    ## From Meeting: Sprint Planning (2026-01-15)

    - [ ] Update API documentation for auth endpoints
    - [ ] Review PRs for user profile feature
    - [ ] Schedule security audit
    ```

=== "Specs"

    Created in `specs/`:

    ```yaml
    ---
    id: api-rate-limiting
    created: 2026-01-15T10:30:00Z
    priority: 1
    status: draft
    ---
    ```

=== "Context"

    Appended to `CONTEXT.md`:

    ```markdown
    ## Architecture Decision: Use Redis for Rate Limiting (2026-01-15)

    We decided to use Redis instead of in-memory storage because...
    ```

### Workflow

1. Provide meeting transcript (paste or file)
2. Coach categorizes: action items → todos, feature requests → specs, decisions → context
3. Coach shows proposed outputs for review
4. You approve
5. Files are created/updated and committed together

---

## Coordinator

Create a dependency-aware work plan with branch assignments.

**CLI:** `spekk coach coordinate`

**Triggers:** "plan the work", "dependency graph", "coordinate development", "organize branches"

### What it does

Analyzes all `draft` and `not_started` assertions, identifies prerequisites, groups related work into feature branches, and updates YAML frontmatter.

### Example output

```
Dependency Analysis
===================

feature/chat-system (4 assertions):
  websocket-connection (no dependencies)
    ↓
  chat-session-model (depends-on: websocket-connection)
    ↓
  chat-message-input (depends-on: chat-session-model)

feature/authentication (2 assertions):
  password-hashing (no dependencies)
  session-tokens (no dependencies)

main (isolated work):
  update-button-styles (no dependencies)
```

### YAML changes

Before:

```yaml
---
id: chat-message-input
created: 2026-01-15T10:00:00Z
priority: 2
status: not_started
---
```

After:

```yaml
---
id: chat-message-input
created: 2026-01-15T10:00:00Z
priority: 2
status: not_started
depends-on: chat-session-model
branch: feature/chat-system
---
```

After updating, the coordinator validates with the parser to catch errors before committing.

---

## Business model validator

Assess startup or business ideas through structured questions.

**CLI:** `spekk coach` (interactive -- trigger by asking)

**Triggers:** "validate business model", "startup validation", "is this viable"

### How it works

1. Asks structured questions (problem, market, solution, competition, business model)
2. Scores responses across dimensions
3. Provides a quantitative health score
4. Identifies risks and opportunities

### Example

```
Business Model Health Score: 72/100

Strengths:
- Clear problem definition (9/10)
- Well-defined target market (8/10)

Risks:
- Competitive landscape is crowded (4/10)
- Unclear path to profitability (5/10)

Recommendations:
1. Focus on differentiation...
2. Validate pricing model...
```

---

## Creating custom skills

Create a markdown file in `specs/coach-skills-system/`:

```markdown
---
id: my-custom-skill
created: 2026-01-15T00:00:00Z
---

# Skill Name

Brief description of what this skill does.

## Triggers

- "keyword 1"
- "keyword 2"

## Workflow

1. First step
2. Second step
3. Third step

## Validation

- Success criterion 1
- Success criterion 2
```

The coach automatically detects and uses any skill file in that directory.

### Tips for good skills

| Do | Don't |
|----|-------|
| Clear, memorable triggers | Vague workflows |
| Step-by-step instructions | Too many steps |
| Concrete validation criteria | Implementation details |
| Examples of expected output | Overlapping triggers with other skills |
