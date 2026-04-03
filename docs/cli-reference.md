---
icon: lucide/square-terminal
---

# CLI Reference

Complete reference for all Spekk CLI commands.

---

## `spekk next`

Show the next priority assertion to work on.

```bash
spekk next                          # Next on current branch
spekk next --all-branches           # Next across all branches
spekk next --spec authentication    # Next in specific spec
spekk next --assertion password-hashing  # Details for specific assertion
spekk next --all                    # Full spec hierarchy (JSON)
```

**Selection logic:**

1. Filters to current git branch (unless `--all-branches`)
2. Removes `done` and `draft` assertions
3. Skips assertions with unfinished dependencies
4. Sorts by priority (1 > 2 > 3)
5. Breaks ties by `created` timestamp (older first)

**Output fields:**

| Field | Description |
|-------|-------------|
| `type` | `"assertion"` or `"spec"` |
| `id` | Assertion identifier |
| `parent` | Parent spec ID |
| `file` | Path to markdown file |
| `priority` | 1 (high), 2 (medium), 3 (low) |
| `status` | Current state |
| `branch` | Assigned branch |
| `dependsOn` | Prerequisite assertion ID |

---

## `spekk builder`

Launch the builder agent to implement assertions.

```bash
spekk builder                       # Loop through all assertions
spekk builder --once                # Build one, then exit
spekk builder --dry-run             # Preview without executing
spekk builder --spec authentication # Only this spec's assertions
spekk builder --assertion login     # Build specific assertion
spekk builder --confirm             # Ask before each build
spekk builder --interactive         # Collaborate with the builder
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| *(none)* | | Loop through all assertions continuously |
| `--once` | | Build one assertion then exit |
| `--dry-run` | `-d` | Preview what would be built |
| `--spec <id>` | `-s` | Only assertions in this spec |
| `--assertion <id>` | | Specific assertion (even if done) |
| `--confirm` | `-c` | Confirm before each build |
| `--interactive` | `-i` | Interactive/collaborative mode |

**How it works:**

1. Gets next priority assertion via parser
2. Reads the assertion requirements
3. Writes tests (when applicable)
4. Implements the feature/fix
5. Runs tests to validate
6. Commits changes
7. Repeats (unless `--once`)

??? info "Lean testing philosophy"

    The builder follows a lean testing approach:

    - Tests behavior, not implementation details
    - One test per meaningful behavior
    - Deletes redundant or low-value tests
    - No tests for trivial code
    - Prefers integration tests over unit when appropriate

---

## `spekk coach`

Launch the coach agent for interactive spec creation.

```bash
spekk coach                          # Interactive mode
spekk coach meeting                  # Process meeting transcript
spekk coach meeting notes.txt        # Process transcript file
spekk coach coordinate               # Plan dependencies
```

### `spekk coach` (interactive)

Helps you write well-formed specifications:

- Asks clarifying questions about requirements
- Creates spec and assertion files
- Ensures correct format and structure
- Commits changes to git

### `spekk coach meeting [file]`

Process meeting transcripts into structured outputs:

- **Todos** → appended to `TODOS.md`
- **Specs** → proper spec files in `specs/`
- **Context** → architectural decisions appended to `CONTEXT.md`

```bash
# Interactive (paste transcript)
spekk coach meeting

# From file
spekk coach meeting standup-notes.txt
```

### `spekk coach coordinate`

Create a dependency-aware work plan:

1. Reads all `draft` and `not_started` assertions
2. Analyzes dependencies
3. Groups related work into feature branches
4. Shows dependency tree for approval
5. Updates YAML frontmatter (`depends-on`, `branch`)
6. Validates with parser
7. Commits changes

---

## `spekk status`

Get a comprehensive overview of all specs.

```bash
spekk status
```

Shows total specs/assertions, status breakdown, completion percentage, and specs grouped by status.

---

## `spekk show`

Launch the interactive web-based spec explorer.

```bash
spekk show
```

Opens a browser with:

- Expandable spec/assertion hierarchy
- Status and priority badges
- Dependency metro map visualization
- Drag-to-pan, click-to-navigate
- Completed specs hidden by default (toggle to show)

---

## `spekk observer`

Monitor spec-code drift.

```bash
spekk observer
```

Detects when code changes but specs don't update (or vice versa). Helps keep specs and implementation synchronized.

---

## `spekk loop`

Run orchestration workflows.

```bash
spekk loop
```

---

## `spekk help`

Show help message.

```bash
spekk help
spekk <command> --help    # Help for specific command
```

---

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Error (check message) |

---

## Environment variables

### `GEMFURY_SPEKK_TOKEN`

Authentication token for the private package registry. Required for installation.

```bash
export GEMFURY_SPEKK_TOKEN=your_token_here
```
