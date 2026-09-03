---
icon: lucide/graduation-cap
---

# Skills

Skills are markdown workflow files that agents read and follow. Both the **coach** and **builder** support skills.

## How skills work

1. **Activation** -- You invoke a skill via the CLI (e.g., `spekk coach meeting`)
2. **Resolution** -- Spekk finds the skill file using layered discovery
3. **Injection** -- The skill content is inlined into the agent's prompt
4. **Execution** -- The agent follows the workflow steps
5. **Validation** -- The agent validates output against success criteria

### Skill discovery

Skills are resolved from three locations, checked in order (first match wins):

| Priority | Location | Scope |
|----------|----------|-------|
| 1 | `.spekk/skills/{agent}/` | **Local** -- project-specific skills |
| 2 | `~/.config/spekk/skills/{agent}/` | **Global** -- your personal skills |
| 3 | Package built-ins | **Default** -- ships with Spekk |

Where `{agent}` is `coach` or `builder`.

A local skill with the same name as a built-in skill will shadow it, letting you customize behavior per-project.

### Skill matching

When you run `spekk coach meeting`, the resolver tries three strategies:

1. **Filename match** -- looks for `meeting.md` in each skill directory
2. **Legacy alias** -- maps `meeting` → `meeting-notes-to-specs-skill.md`
3. **Frontmatter ID** -- scans all `.md` files for a `id: meeting` field in YAML frontmatter

---

## Built-in coach skills

### Meeting notes to specs

Process meeting transcripts into structured outputs.

**CLI:** `spekk coach meeting [file]`

**Aliases:** `meeting` → `meeting-notes-to-specs-skill`

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

#### Workflow

1. Provide meeting transcript (paste or file)
2. Coach categorizes: action items → todos, feature requests → specs, decisions → context
3. Coach shows proposed outputs for review
4. You approve
5. Files are created/updated and committed together

---

### Coordinator

Create a dependency-aware work plan with branch assignments.

**CLI:** `spekk coach coordinate`

**Aliases:** `coordinate` → `coordinator-skill`

#### What it does

Analyzes all `draft` and `not_started` assertions, identifies prerequisites, groups related work into feature branches, and updates YAML frontmatter.

#### Example output

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

#### YAML changes

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

### Business model validator

Assess startup or business ideas through structured questions.

**CLI:** `spekk coach validate`

**Aliases:** `validate` → `business-model-validator-skill`

**Triggers:** "validate business model", "startup validation", "is this viable"

#### How it works

1. Asks structured questions (problem, market, solution, competition, business model)
2. Scores responses across dimensions
3. Provides a quantitative health score
4. Identifies risks and opportunities

#### Example

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

## Built-in builder skills

### Review

Review what was just built against the assertions marked `done`. The review fixes what it finds, and the push waits for it.

**CLI:** `spekk builder review` (a fresh session) or `spekk skill show builder review` (adopt it in the current session)

**Alias:** `review` → `review-skill`

**Scope:** the assertions marked `done` on the current branch since it left its base branch, plus the diff from that base to `HEAD`. On the base branch itself, you name the commit range.

**Lenses, in order, each with a remedy:**

1. Every success criterion of every in-scope assertion is checked against the real code. An unmet one is fixed, or the assertion is set to `failed`.
2. Every test earns its place. A test that passes when its behavior is broken, restates the implementation, duplicates another test, or exercises a mock is deleted.
3. Nothing beyond what the assertions ask for. Unrequested generality, configuration, and abstraction are removed. A hunk no assertion accounts for is reverted or explained.
4. Errors are loud. A dropped, defaulted, or broadly caught error is fixed.
5. The spec tree is sound: `spekk validate` and `spekk next` succeed, no stale lock, every `**Tests:**` link resolves.
6. The diff is fit to publish: no secret, no private name, no reference to another repository.

The review writes no observation file. Its output is the fixes in the working tree and a short report in the session: each assertion with a verdict, what was fixed, what was deleted, and what stays open.

The `spekk-dev-loop` skill loads this skill in its verify phase.

## Creating custom skills

Create a markdown file in your local or global skills directory:

```bash
# Local (this project only)
.spekk/skills/coach/my-skill.md

# Global (all projects)
~/.config/spekk/skills/coach/my-skill.md

# Builder skills work the same way
.spekk/skills/builder/my-skill.md
```

### Skill file format

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

Once created, invoke it directly:

```bash
spekk coach my-skill
spekk builder my-skill
```

### Tips for good skills

| Do | Don't |
|----|-------|
| Clear, memorable triggers | Vague workflows |
| Step-by-step instructions | Too many steps |
| Concrete validation criteria | Implementation details |
| Examples of expected output | Overlapping triggers with other skills |
