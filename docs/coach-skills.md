---
icon: lucide/graduation-cap
---

# Skills

A skill is a markdown file with a workflow that an agent reads and follows. The coach, the builder, and the observer all support skills.

## How skills work

1. **Activation.** You name the skill on the command line, for example `spekk coach meeting`.
2. **Resolution.** Spekk finds the skill file through the layers below.
3. **Injection.** Spekk inlines the skill content into the agent's prompt.
4. **Execution.** The agent follows the workflow steps.
5. **Validation.** The agent checks its output against the skill's success criteria.

### Skill discovery

Spekk looks in three places, in order. The first match wins:

| Order | Location | Scope |
|-------|----------|-------|
| 1 | `.spekk/skills/{agent}/` | Local: skills for this project |
| 2 | `~/.config/spekk/skills/{agent}/` | Global: your skills, for every project |
| 3 | The skills in the binary | Default: what ships with Spekk |

`{agent}` is `coach`, `builder`, or `observer`.

A local skill with the same name as a built-in skill shadows it, so you can change a skill's behavior per project.

### Skill matching

When you run `spekk coach meeting`, the resolver tries three things in each directory:

1. **File name.** A file named `meeting.md`.
2. **Alias.** `meeting` maps to `meeting-notes-to-specs-skill.md`. The aliases are `meeting`, `coordinate`, and `validate` for the coach, and `coverage-gap` and `prune` for the observer.
3. **Frontmatter id.** Any `.md` file with `id: meeting` in its YAML frontmatter.

## Built-in coach skills

### Meeting notes to specs

Turn a meeting transcript into three outputs.

**Command:** `spekk coach meeting [file]`

**Alias:** `meeting` maps to `meeting-notes-to-specs-skill`

=== "Todos"

    Appended to `TODOS.md`:

    ```markdown
    ## From Meeting: Sprint Planning (2026-01-15)

    - [ ] Update API documentation for auth endpoints
    - [ ] Review PRs for user profile feature
    - [ ] Schedule security audit
    ```

=== "Specs"

    Written to `specs/`:

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

1. Give the coach the transcript, as a file or pasted text.
2. The coach sorts the content: action items become todos, feature requests become specs, decisions become context.
3. The coach shows the proposed output.
4. You approve it.
5. The coach writes the files and commits them together.

### Coordinator

Make a dependency-aware work plan with branch assignments.

**Command:** `spekk coach coordinate`

**Alias:** `coordinate` maps to `coordinator-skill`

The coordinator reads every `draft` and `not_started` assertion, finds the prerequisites, groups related work into feature branches, and writes the result into the frontmatter.

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

#### Frontmatter changes

Before:

```yaml
---
id: chat-message-input
parent: chat
created: 2026-01-15T10:00:00Z
priority: 2
status: not_started
---
```

After:

```yaml
---
id: chat-message-input
parent: chat
created: 2026-01-15T10:00:00Z
priority: 2
status: not_started
depends-on: chat-session-model
branch: feature/chat-system
---
```

After the update, the coordinator runs `spekk validate` to catch a fault before it commits.

### Business model validator

Assess a business idea through structured questions.

**Command:** `spekk coach validate`

**Alias:** `validate` maps to `business-model-validator-skill`

**Triggers:** "validate business model", "startup validation", "is this viable"

The coach asks about the problem, the market, the solution, the competition, and the business model. It scores each answer, gives a total score, and lists risks and opportunities.

#### Example

```
Business Model Health Score: 72/100

Strengths:
- Clear problem definition (9/10)
- Well-defined target market (8/10)

Risks:
- Many competitors (4/10)
- Unclear path to profitability (5/10)

Recommendations:
1. Focus on differentiation...
2. Validate pricing model...
```

## Built-in observer skills

| Skill | Command | What it does |
|-------|---------|--------------|
| Coverage gap | `spekk observer coverage-gap` | Finds code that a spec could document. An aid for gradual adoption, not a defect report |
| Prune | `spekk observer prune` | Finds code that nothing uses, and design-level redundancy. It recommends and never deletes |
| Consolidate | `spekk observer consolidate` | Curates the open observations by editing their frontmatter on their own branches |

See [`spekk observer`](cli-reference.md#spekk-observer) for the details.

### Property tests

Decide whether a promise deserves a property-based test, then write it for the right layer and prove it reached the state it guards.

**CLI:** `spekk coach property-tests [assertion-id]`

**Aliases:** `properties` → `property-tests-skill`

**Triggers:** "property test", "add a property", "cover this assertion with a property", "false positive in the sweep"

#### How it works

1. Finds the `done` assertions a property could restate, with `spekk status` and `spekk query`
2. Applies a value gate before any code: the promise is `done`, needs search, would matter if broken, has evidence, costs less than it is worth, and keeps the portfolio balanced
3. Studies the code through fixed lenses for both layers, a browser explorer and a backend property library
4. Writes the catalog entry, chooses the form, implements in the project's house pattern using only the installed tool version's API
5. Runs the property clean against seeded data, proves the run reached the state, and files one issue per surviving violation with a strict expected failure that names it

#### Example

```
Property catalog entry

Name: Next visit implies an open case
Invariant: a non-empty Next Visit cell means Open Cases is at least 1
Assertion: patient-list-enriched-columns (done)
Value 4 / Cost 1
Form: always(...) over an extractor that returns the rows as JSON
```

---

## Creating custom skills

Write a markdown file in a local or global skills directory:

```bash
.spekk/skills/coach/my-skill.md             # This project only
~/.config/spekk/skills/coach/my-skill.md    # Every project
.spekk/skills/builder/my-skill.md           # A builder skill
.spekk/skills/observer/my-skill.md          # An observer skill
```

### Skill file format

```markdown
---
id: my-custom-skill
created: 2026-01-15T00:00:00Z
---

# Skill Name

What this skill does, in one or two sentences.

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

Then invoke it:

```bash
spekk coach my-skill
spekk builder my-skill
spekk observer my-skill
```

`spekk install <agent> <skill>` fetches a skill from the registry into one of these directories. See [Installing skills](cli-reference.md#spekk-install-agent-skill).

### Tips for good skills

| Do | Do not |
|----|--------|
| Clear triggers that a person remembers | Vague workflows |
| Step-by-step instructions | Too many steps |
| Concrete validation criteria | Implementation details |
| Examples of the expected output | Triggers that overlap with another skill |
