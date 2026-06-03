---
icon: lucide/lightbulb
---

# Concepts

The core ideas behind spec-driven development with Spekk.

## Specs and assertions

### Specs

A **spec** defines a feature or capability. It lives in `specs/<name>/<name>.md` and contains a high-level description of what you're building.

```
specs/
├── authentication/
│   ├── authentication.md          # The spec
│   └── assertions/                # What must be true
│       ├── password-hashing.md
│       └── session-tokens.md
```

A spec's status is **computed automatically** from its assertions:

- All assertions `done` → spec is `done`
- Any assertion `in_progress` → spec is `in_progress`
- Otherwise → `not_started`

### Assertions

An **assertion** is an atomic unit of work. Each assertion is a markdown file with YAML frontmatter:

```markdown
---
id: password-hashing
created: 2026-01-21T12:00:00Z
priority: 1
status: not_started
---

# Passwords Must Be Hashed

## What Must Be True

All user passwords are hashed with bcrypt before storage.
Plain-text passwords are never written to the database.

## Success Criteria

- Passwords are hashed with bcrypt (cost factor 12)
- Login compares against hashed value
- No plain-text passwords in logs or database
```

!!! tip "One assertion per file"
    Keep assertions atomic. Each file describes one testable behavior. If you find yourself writing "and" in the title, split it into two assertions.

---

## Frontmatter fields

### Required fields

| Field | Description | Example |
|-------|-------------|---------|
| `id` | Unique identifier | `password-hashing` |
| `created` | ISO timestamp (never changes) | `2026-01-21T12:00:00Z` |
| `priority` | 1 (highest) to 3 (lowest) | `1` |
| `status` | Current state | `not_started` |

### Optional fields

| Field | Description | Example |
|-------|-------------|---------|
| `depends-on` | ID of prerequisite assertion | `websocket-connection` |
| `branch` | Git branch for this work | `feature/chat-system` |

### Status values

| Status | Meaning |
|--------|---------|
| `not_started` | Default. Not yet implemented. |
| `in_progress` | Currently being worked on. |
| `done` | Fully implemented and validated. |
| `draft` | Placeholder/planning. Excluded from work queue. |
| `failed` | Confirmed issue that needs fixing. |

### Priority levels

| Priority | Meaning |
|----------|---------|
| **1** | Highest -- critical, blocking |
| **2** | Medium -- important, not blocking |
| **3** | Lowest -- nice to have, future |

!!! note "Only three levels"
    Intentionally limited to force clear prioritization. If everything is priority 1, nothing is.

---

## Dependencies

Assertions can declare a single dependency using `depends-on`:

```yaml
---
id: chat-message-input
depends-on: chat-session-model
branch: feature/chat-system
---
```

**Rules:**

- Each assertion can depend on **at most one** other assertion
- The parser validates that dependency IDs exist
- `spekk next` skips assertions whose dependencies aren't `done`
- No circular dependencies allowed

**If you need multiple prerequisites**, either:

- **Sequence them:** A → B → C (chain of single dependencies)
- **Create a junction:** A "prerequisites-ready" assertion that the others depend on

---

## Branches

Assertions can be assigned to git branches:

```yaml
---
id: password-hashing
branch: feature/authentication
---
```

- Defaults to `main` if omitted
- `spekk next` filters to the current branch by default
- Use `spekk next --all-branches` to see everything
- Assertions on different branches should be independent (buildable in parallel)

---

## Priority rules

When determining what to build next, the parser follows these rules:

1. **Filter** -- Remove `done`, `draft`, and blocked assertions
2. **Branch** -- Only show assertions for the current git branch
3. **Priority** -- Lower number = higher priority (1 beats 2 beats 3)
4. **Tiebreak** -- Oldest `created` timestamp wins

---

## Writing good specs

### Do

- **Be testable** -- Clear success criteria that can be verified
- **Be atomic** -- One assertion per file, one behavior per assertion
- **Be ordered** -- Priority indicates sequence of importance
- **Be immutable** -- `created` timestamp never changes
- **Be clear** -- Anyone should understand the requirements

### Don't

- **Be vague** -- "Make it better" is not a spec
- **Be compound** -- Multiple concerns in one assertion
- **Be prescriptive** -- Say WHAT, not HOW (leave implementation to the builder)
- **Be indecisive** -- Avoid changing priorities mid-work

---

## The development loop

``` mermaid
graph TD
  A["spekk coach"] -->|creates| B["Specs & Assertions"];
  B -->|prioritized by| C["spekk next"];
  C -->|built by| D["spekk builder"];
  D -->|validates| E["Tests pass"];
  E -->|updates| F["Status: done"];
  F -->|next assertion| C;
```

1. **Coach creates specs** from your requirements
2. **Parser prioritizes** the next assertion
3. **Builder implements** the assertion
4. **Tests validate** the implementation
5. **Status updates** to done
6. **Repeat** until all assertions are complete
