# Spekk CLI Reference

Complete reference for all Spekk CLI commands.

## Global Commands

### `spekk`

Alias for `spekk next` - shows the next priority assertion to work on.

```bash
spekk
```

## Coach Commands

The coach helps you create and manage specifications.

### `spekk coach`

Launch interactive coach for spec creation.

```bash
spekk coach
```

**What it does:**
- Helps you write well-formed specifications
- Asks clarifying questions about requirements
- Creates spec and assertion files
- Commits changes to git

**When to use:**
- Creating new features
- Refining vague requirements
- Breaking down large projects

---

### `spekk coach meeting [file]`

Process meeting transcripts into structured outputs.

```bash
# Interactive mode (paste transcript)
spekk coach meeting

# Process transcript file
spekk coach meeting notes.txt
spekk coach meeting standup-2024-01-15.md
```

**What it extracts:**
- **Todos** → Appended to `TODOS.md`
- **Specs** → Proper spec files in `specs/`
- **Context** → Architectural decisions appended to `CONTEXT.md`

**Workflow:**
1. Provide meeting transcript
2. Coach extracts structured data
3. Shows proposed outputs for approval
4. Creates files and commits together

**Input formats:**
- Plain text
- Markdown
- Any transcript format

---

### `spekk coach coordinate`

Analyze assertions and create dependency-aware work plan.

```bash
spekk coach coordinate
```

**What it does:**
1. Reads all `draft` and `not_started` assertions
2. Analyzes dependencies (what needs to be built first)
3. Groups related assertions into feature branches
4. Shows dependency tree for approval
5. Updates YAML frontmatter with:
   - `depends-on: assertion-id` (single parent dependency)
   - `branch: feature/name` (branch assignment)
6. Validates with parser
7. Commits changes

**Output example:**
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
```

**When to use:**
- Planning parallel development
- Understanding build order
- Scoping work to branches
- Before starting implementation

---

## Parser Commands

View and query specifications.

### `spekk next`

Show next priority assertion to work on.

```bash
# Next assertion on current branch
spekk next

# Next assertion across all branches
spekk next --all-branches

# Next assertion in specific spec
spekk next --spec authentication

# Details for specific assertion
spekk next --assertion password-hashing

# Full spec hierarchy (JSON)
spekk next --all
```

**Selection logic:**
1. Filters to current git branch (unless `--all-branches`)
2. Filters out `done` assertions
3. Filters by dependencies (only returns if deps are done)
4. Sorts by priority (1 > 2 > 3)
5. Breaks ties by creation timestamp (older first)

**Fields shown:**
- `type` - "assertion" or "spec"
- `id` - Assertion identifier
- `parent` - Parent spec ID
- `file` - Path to markdown file
- `priority` - 1 (high), 2 (medium), or 3 (low)
- `status` - Current state
- `branch` - Assigned branch
- `dependsOn` - Parent dependency (if any)

---

### `spekk status`

Get comprehensive overview of all specs.

```bash
spekk status
```

**Shows:**
- Total specs and assertions
- Status breakdown (done, in-progress, not started)
- Completion percentage
- Specs grouped by status

---

### `spekk show`

Launch interactive web-based spec explorer.

```bash
spekk show
```

Opens browser with navigable spec tree.

---

## Builder Commands

Automate implementation of specifications.

### `spekk builder`

Build assertions automatically.

```bash
# Loop through all assertions
spekk builder

# Build one assertion then exit
spekk builder --once

# Preview what would be built (no execution)
spekk builder --dry-run

# Work only on specific spec
spekk builder --spec authentication

# Build specific assertion (even if done)
spekk builder --assertion password-hashing

# Supervised mode (confirm before each build)
spekk builder --confirm

# Interactive mode (collaborate with builder)
spekk builder --interactive
```

**How it works:**
1. Gets next priority assertion via parser
2. Reads assertion requirements
3. Writes tests (when applicable)
4. Implements the feature/fix
5. Runs tests to validate
6. Commits changes
7. Repeats (unless `--once`)

**Builder flags:**

| Flag | Description |
|------|-------------|
| *(none)* | Loop through all assertions continuously |
| `--once` | Build one assertion then exit |
| `--dry-run`, `-d` | Preview what would be built, don't execute |
| `--spec <id>`, `-s <id>` | Work only on assertions in this spec |
| `--assertion <id>` | Work on specific assertion (even if done) |
| `--confirm`, `-c` | Ask y/n before each build |
| `--interactive`, `-i` | Start builder in interactive mode |

---

## Observer Commands

Monitor spec-code drift.

### `spekk observer`

Detect when code changes but specs don't (or vice versa).

```bash
spekk observer
```

Helps keep specs and implementation synchronized.

---

## Loop Commands

Run orchestration workflows.

### `spekk loop`

Execute workflow orchestration.

```bash
spekk loop
```

*(Advanced feature - see orchestration docs)*

---

## Help Commands

### `spekk help`

Show general help message.

```bash
spekk help
```

### `spekk <command> --help`

Show help for specific command.

```bash
spekk coach --help
spekk builder --help
spekk next --help
```

---

## Exit Codes

- `0` - Success
- `1` - Error (check error message)

---

## Environment Variables

### `GEMFURY_SPEKK_TOKEN`

Authentication token for private package registry.

```bash
export GEMFURY_SPEKK_TOKEN=your_token_here
```

Required for installation from private registry.

---

## Common Workflows

### Creating a New Feature

```bash
# 1. Define the spec
spekk coach
> "Add user authentication"

# 2. Coordinate dependencies
spekk coach coordinate

# 3. Create feature branch
git checkout -b feature/authentication

# 4. Build the spec
spekk builder --once

# 5. Commit and repeat
```

### Processing Meeting Notes

```bash
# After a meeting
spekk coach meeting meeting-notes.txt

# Review extracted todos
cat TODOS.md

# Review new specs
spekk next --all
```

### Planning a Sprint

```bash
# See what's ready to build
spekk status

# Coordinate work across branches
spekk coach coordinate

# Create branches and assign to team
git checkout -b feature/chat-system
git checkout -b feature/auth-system
```

---

## Tips

- **Use `--dry-run`** to preview builder actions before executing
- **Use `--confirm`** for supervised building (review before each step)
- **Use `--once`** to build one assertion at a time
- **Use `spekk coordinate`** before starting parallel development
- **Commit specs separately** from implementation (easier to review)

---

## See Also

- [Getting Started Guide](./GETTING-STARTED.md)
- [Coach Skills Guide](./COACH-SKILLS.md)
- [Spec Format Reference](../README.md#spec-format)
