# Spekk CLI

**Spec-driven development CLI tool** for managing specifications and automating development workflows with AI agents.

## Overview

Spekk CLI enables spec-driven development by:
- **Parsing** specification files to identify what needs to be built
- **Orchestrating** AI agents (Coach and Builder) to create and implement specs
- **Automating** the development workflow loop

## Documentation

- **[Getting Started Guide](./docs/GETTING-STARTED.md)** - Quick introduction to Spekk
- **[CLI Reference](./docs/CLI-REFERENCE.md)** - Complete command documentation
- **[Coach Skills Guide](./docs/COACH-SKILLS.md)** - Detailed skill workflows
- **[Release Notes](./docs/RELEASE-NOTES-1.1.0.md)** - Version history

## Installation

### 1. Get a GemFury Token

Ask a team lead for a GemFury **read** (or full access) token for the `thinknimble` org.

### 2. Add the Token to Your Shell

```bash
# Add to ~/.zshrc or ~/.bashrc
export GEMFURY_SPEKK_TOKEN=your_token_here
```

Then reload your shell config:
```bash
source ~/.zshrc  # or ~/.bashrc
```

### 3. Configure npm

Add to your global `~/.npmrc`:
```
@spekk:registry=https://npm.fury.io/thinknimble/
//npm.fury.io/thinknimble/:_authToken=${GEMFURY_SPEKK_TOKEN}
```

Or run these commands:
```bash
npm config set @spekk:registry https://npm.fury.io/thinknimble/
npm config set //npm.fury.io/thinknimble/:_authToken "\$GEMFURY_SPEKK_TOKEN"
```

### 4. Install

```bash
npm install -g @spekk/cli
```

### 5. Verify

```bash
spekk --help
```

### Updating

```bash
npm update -g @spekk/cli
```

## Quick Start

```bash
# Find next priority assertion to work on
spekk next

# Launch builder (implements specs continuously)
spekk builder

# Launch builder for one assertion then exit
spekk builder --once

# Launch coach (creates specs interactively)
spekk coach

# Process meeting notes into specs/todos/context
spekk coach meeting notes.txt

# View all specs in interactive web UI
spekk show

# Get comprehensive overview of all specs
spekk status
```

## How It Works

### 1. Spec Structure

Specifications live in `specs/` directory with this structure:

```
specs/
├── spec-parser/
│   ├── spec-parser.md           # Main spec
│   └── assertions/              # What must be true
│       ├── parses-frontmatter.md
│       ├── validates-fields.md
│       └── ...
├── coach-agent/
│   ├── coach-agent.md
│   ├── coach-agent.prompt.md    # Agent instructions
│   └── assertions/
└── ...
```

### 2. Spec Format

Each spec/assertion is a markdown file with YAML frontmatter:

```markdown
---
id: my-feature
created: 2026-01-21T12:00:00Z
priority: 1                   # 1=highest, 2=medium, 3=lowest
status: not_started          # not_started | in_progress | done | draft
---

# Feature Name

## What Must Be True

Describe what needs to exist/work for this to be complete.

## Success Criteria

- ✅ Specific, testable criteria
- ✅ Clear validation steps
```

**Parent spec status is computed** — a parent spec's status automatically reflects its assertions' states (all done = done, any in-progress = in-progress, etc.).

### Optional Fields

**`depends-on`**: Single assertion ID that must be completed before this one.
- Creates linear dependency chains
- Parser validates reference exists
- `spekk next` respects dependencies

**`branch`**: Git branch where this assertion lives.
- Defaults to `main` if omitted
- `spekk next` filters to current branch by default
- Use `spekk next --all-branches` to see all assertions

```yaml
---
id: feature-x
depends-on: prerequisite-y  # Must complete prerequisite-y first
branch: feature/my-feature   # Lives on feature branch
---
```

### 3. The Parser

The spec parser (`src/parser/`) reads all specs and identifies the next priority work item:

```bash
$ spekk next
{
  "type": "assertion",
  "id": "enforces-folder-structure",
  "parent": "spec-parser",
  "file": "specs/spec-parser/assertions/enforces-folder-structure.md",
  "priority": 1,
  "status": "not_started",
  "title": "Parser Must Enforce Folder Structure"
}
```

**Priority rules:**
1. Lower number = higher priority (1 > 2 > 3)
2. Same priority? Oldest `created` timestamp wins
3. Excludes `done` and `draft` statuses

**Filtering:**
```bash
# Get next assertion from a specific spec
spekk next --spec auth

# Get details for a specific assertion
spekk next --assertion login-button

# View full spec hierarchy
spekk next --all
```

### 4. Agent Workflows

#### Builder Agent

Automates implementation of specs:

```bash
# Loop through all assertions continuously (default)
spekk builder

# Build one assertion then exit
spekk builder --once

# Preview what would be built without launching Claude
spekk builder --dry-run

# Work only on assertions in a specific spec
spekk builder --spec auth

# Build a specific assertion (even if already done)
spekk builder --assertion login-button

# Supervised mode: confirm before each build
spekk builder --confirm

# Interactive mode: collaborate with the builder
spekk builder --interactive
```

**How it works:**
1. Gets next priority assertion via parser
2. Reads the assertion requirements
3. Writes tests (when applicable)
4. Implements the feature/fix
5. Runs tests to validate
6. Commits changes
7. Repeats until all assertions done (or `--once` flag set)

**Lean Testing Philosophy**

The builder follows a lean testing approach:
- Tests behavior, not implementation details
- One test per meaningful behavior
- Deletes redundant or low-value tests
- No tests for trivial code
- Prefers integration tests over unit when appropriate

**Result:** Fast, trustworthy test suite focused on real behavior validation.

#### Coach Agent

Helps create well-formed specs:

```bash
# Launch interactive coach
spekk coach

# Process meeting transcript into specs/todos/context
spekk coach meeting

# Process a transcript file directly
spekk coach meeting notes.txt
```

**Meeting Processing Mode:**

The coach can extract structured outputs from meeting transcripts:
- **Todos** → appended to `TODOS.md`
- **Specs** → proper spec files in `specs/`
- **Context** → architectural decisions appended to `CONTEXT.md`

All outputs are proposed before creation, then committed together in a single commit.

**Interactive Mode:**

The coach:
1. Takes user feature requests
2. Asks clarifying questions
3. Creates detailed spec files
4. Ensures specs follow format
5. Commits new specs

#### Observer Agent

Monitors spec-code drift and detects when code changes but specs don't (or vice versa):

```bash
spekk observer
```

The observer helps keep specs and implementation synchronized.

## Commands

### Core Commands

```bash
spekk              # Default: runs parser (equivalent to `spekk next`)
spekk next         # Get next priority assertion
spekk show         # Launch interactive web spec explorer
spekk status       # Comprehensive overview of all specs/assertions
spekk coach        # Launch Coach Agent
spekk builder      # Launch Builder Agent
spekk observer     # Launch Observer Agent (monitors drift)
spekk loop         # Run orchestration workflows
spekk help         # Show help message
```

### Builder Flags

| Flag | Description |
|------|-------------|
| *(none)* | Loop through all assertions continuously (default) |
| `--once` | Build one assertion then exit |
| `--dry-run`, `-d` | Preview what would be built, don't launch Claude |
| `--spec <id>`, `-s <id>` | Work only on assertions in this spec |
| `--assertion <id>` | Work on a specific assertion (even if done) |
| `--confirm`, `-c` | Ask y/n before each build |
| `--interactive`, `-i` | Start builder in interactive mode |

### Coach Subcommands

```bash
spekk coach                   # Interactive spec creation
spekk coach meeting           # Meeting processing (prompts for transcript)
spekk coach meeting notes.txt # Process transcript file directly
```

### Parser Flags

```bash
spekk next                    # Get next priority assertion
spekk next --all              # Get full spec hierarchy (JSON)
spekk next --spec <id>        # Filter to assertions in a specific spec
spekk next --assertion <id>   # Get details for specific assertion
```

## Directory Structure

```
spekk-cli/
├── src/                    # All implementation code
│   ├── parser/            # Spec parser
│   │   ├── index.js       # Core logic
│   │   └── cli.js         # CLI interface
│   ├── coach/             # Coach agent
│   │   ├── cli.js
│   │   └── meeting-notes-to-specs.js
│   ├── builder/           # Builder agent
│   │   └── cli.js
│   └── observer/          # Observer agent
│       └── cli.js
├── bin/
│   └── spekk.js          # Main CLI entry point
├── specs/                 # All specifications
│   ├── spec-parser/
│   ├── coach-agent/
│   ├── builder-agent/
│   └── spekk-cli/
├── docs/
│   └── RELEASE-NOTES-1.1.0.md
└── package.json
```

## Development Workflow

### Creating New Features

1. **Write the spec** (or use coach agent):
   ```bash
   spekk coach
   ```

2. **Implement via builder**:
   ```bash
   spekk builder
   ```

3. **Or implement manually**:
   ```bash
   # Check what's next
   spekk next

   # Read the assertion file
   cat specs/spec-parser/assertions/parses-frontmatter.md

   # Implement the feature
   # ... write code ...

   # Update status to done
   # Edit frontmatter: status: not_started → status: done
   ```

### Processing Meeting Notes

```bash
# Launch coach in meeting mode
spekk coach meeting

# Or process a file directly
spekk coach meeting standup-notes.txt
```

The coach will:
1. Extract todos, specs, and context
2. Show you what it will create
3. Wait for approval
4. Create files and commit together

### Spec Best Practices

**Good specs are:**
- ✅ **Testable** - Clear success criteria
- ✅ **Atomic** - One assertion per file
- ✅ **Ordered** - Priority indicates sequence
- ✅ **Immutable** - `created` timestamp never changes
- ✅ **Clear** - Anyone can understand requirements

**Avoid:**
- ❌ Vague requirements ("make it better")
- ❌ Multiple concerns in one assertion
- ❌ Implementation details (say WHAT, not HOW)
- ❌ Changing priorities mid-work

## Status Values

- **`not_started`** - Default, not yet implemented
- **`in_progress`** - Currently being worked on
- **`done`** - Fully implemented and validated
- **`draft`** - Placeholder/planning (excluded from work queue)
- **`failed`** - Confirmed implementation issue that needs fixing

## Priority Levels

- **1** - Highest priority (critical, blocking)
- **2** - Medium priority (important, not blocking)
- **3** - Lowest priority (nice to have, future)

Keep it simple: only 3 levels. Forces clear prioritization decisions.

## Testing

```bash
# Run all tests
npm test

# Run implementation tests only
npm run test:impl

# Run spec validation tests only
npm run test:specs
```

The spec system follows a **lean testing philosophy**:
- Tests validate behavior, not implementation details
- One test per meaningful behavior (avoid redundancy)
- No tests for trivial code (getters, simple pass-throughs)
- Integration tests over unit tests when appropriate
- Fast, trustworthy suite over maximum coverage

## Requirements

- **Node.js** 18+ (ES modules support)
- **Claude CLI** (for builder/coach/observer agents)
- **Git** (for automated commits)

## Philosophy

**Spec-driven development** means:
1. Write specs first (what must be true)
2. Let AI agents implement (how to make it true)
3. Validate with tests (prove it's true)
4. Iterate continuously (keep specs updated)

The specs are the source of truth. Code is the implementation of specs. Tests prove specs are satisfied.

## What's New in 1.1.0

### Builder CLI Flags
- Builder now loops continuously by default (use `--once` to stop after one)
- New flags: `--dry-run`, `--spec`, `--assertion`, `--confirm`, `--interactive`
- Parser supports `--spec` and `--assertion` filtering

### Meeting Processing
- `spekk coach meeting [file]` extracts todos/specs/context from transcripts
- Single-commit workflow with approval step
- Auto-detects meeting keywords in regular coach sessions

### Lean Testing
- Builder enforces lean testing philosophy
- 186 tests → 116 tests (suite runs in ~300ms, was ~830ms)
- Focus on meaningful behavior validation

### Other Improvements
- Parent spec status automatically computed from assertions
- Browser suppression during tests for cleaner output

See `docs/RELEASE-NOTES-1.1.0.md` for full details.

## More Information

- **Main spec**: `specs/spec-parser/spec-parser.md`
- **Coach agent**: `specs/coach-agent/coach-agent.prompt.md`
- **Builder agent**: `specs/builder-agent/builder-agent.prompt.md`
- **CLI spec**: `specs/spekk-cli/spekk-cli.md`

## License

MIT
