---
icon: lucide/lightbulb
---

# Concepts

The ideas behind spec-driven development with Spekk.

## Specs and assertions

### Specs

A **spec** describes one feature or capability. It lives in `specs/<name>/<name>.md` and says, at a high level, what you are building.

```
specs/
├── authentication/
│   ├── authentication.md          # The spec
│   └── assertions/                # What must be true
│       ├── password-hashing.md
│       └── session-tokens.md
```

Spekk computes a spec's status from its assertions:

- Draft assertions do not count.
- Any `failed` assertion makes the spec `failed`.
- All assertions `done` makes the spec `done`.
- A spec with no assertions, or with only draft assertions, is `not_started`.
- Every other mix is `in_progress`.

Do not write a `status` on a spec. The one exception is the literal `draft`, which hides the spec and all of its assertions from the work queue.

### Assertions

An **assertion** is one small unit of work. Each assertion is a markdown file with YAML frontmatter:

```markdown
---
id: password-hashing
parent: authentication
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
    Keep each assertion small. Each file describes one testable behavior. When the title needs the word "and", split it into two assertions.

## Frontmatter fields

### Required fields

| Field | Description | Example |
|-------|-------------|---------|
| `id` | Unique identifier, kebab-case | `password-hashing` |
| `parent` | The id of the spec. Assertions only | `authentication` |
| `created` | ISO 8601 timestamp. It never changes | `2026-01-21T12:00:00Z` |
| `priority` | 1 (highest) to 3 (lowest) | `1` |

### Optional fields

| Field | Description | Example |
|-------|-------------|---------|
| `status` | Current state. Default: `not_started`. On a spec, only the literal `draft` is allowed | `in_progress` |
| `branch` | Git branch for this work. Default: `main` | `feature/chat-system` |
| `depends-on` | Id of one assertion that must be `done` first. Assertions only | `websocket-connection` |
| `locked-by` | Set by a builder while it holds an `in_progress` assertion. Removed when the status changes | `builder-myhost-4242-1756562531` |

A key outside this set is a custom field. `spekk validate` accepts it, and `spekk query` can read it. See [Custom frontmatter fields](cli-reference.md#custom-frontmatter-fields).

### Status values

| Status | Meaning |
|--------|---------|
| `not_started` | The default. Not yet implemented. |
| `in_progress` | Somebody is working on it. |
| `done` | Implemented and validated. |
| `draft` | A placeholder. Out of the work queue. |
| `failed` | A confirmed fault that needs a fix. |

### Priority levels

| Priority | Meaning |
|----------|---------|
| **1** | Highest. Critical, or it blocks other work. |
| **2** | Medium. Important, not blocking. |
| **3** | Lowest. Good to have, later. |

!!! note "Three levels only"
    The limit forces a clear order. When everything is priority 1, nothing is.

## Dependencies

An assertion can declare one dependency with `depends-on`:

```yaml
---
id: chat-message-input
parent: chat
depends-on: chat-session-model
branch: feature/chat-system
---
```

Rules:

- Each assertion depends on at most one other assertion.
- `spekk validate` checks that the id exists and that there is no cycle.
- `spekk next` skips an assertion whose dependency is not `done`.

When an assertion needs more than one prerequisite, do one of two things:

- **Make a chain.** A, then B, then C, each with one `depends-on`.
- **Make a junction.** One "prerequisites-ready" assertion that the others depend on.

A list in `depends-on` is refused, and the error stops the parse of the whole tree. The single id is what orders the work: a chain releases assertions one at a time, and assertions with no link between them can run in parallel. The coach's `coordinate` skill turns a set of prerequisites into a chain.

## Branches

An assertion can name a git branch:

```yaml
---
id: password-hashing
parent: authentication
branch: feature/authentication
---
```

- The default is `main`.
- `spekk next` shows the current branch only, by default.
- `spekk next --all-branches` shows every branch.
- Assertions on different branches must be independent, so that they can be built in parallel.

## Selection rules

`spekk next` picks the next assertion in this order:

1. **Filter.** Drop `done` and `draft` assertions, every assertion of a `draft` spec, every assertion whose dependency is not `done`, and every `in_progress` assertion that a builder holds with a fresh lock.
2. **Branch.** Keep the current git branch only.
3. **Priority.** A lower number wins: 1 before 2 before 3.
4. **Tiebreak.** The oldest `created` timestamp wins.

## Writing good specs

### Do

- **Be testable.** Write success criteria that a test can check.
- **Be small.** One assertion per file, one behavior per assertion.
- **Be ordered.** The priority says what comes first.
- **Be stable.** The `created` timestamp never changes.
- **Be clear.** A new reader must understand the requirement.

### Do not

- **Be vague.** "Make it better" is not a spec.
- **Be compound.** Keep one concern per assertion.
- **Be prescriptive.** Say what, not how. The builder decides how.
- **Be indecisive.** Do not change priorities in the middle of the work.

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

1. The coach writes specs from your requirements.
2. `spekk next` picks the next assertion.
3. The builder implements it.
4. The tests validate the implementation.
5. The status becomes `done`.
6. Repeat until every assertion is done.
