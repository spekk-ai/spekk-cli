# Spekk CLI

**Spec-driven development CLI tool** for managing specifications and automating development workflows with AI agents.

## Overview

Spekk CLI enables spec-driven development by:
- **Parsing** specification files to identify what needs to be built
- **Orchestrating** AI agents (Coach and Builder) to create and implement specs
- **Automating** the development workflow loop

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
# Install dependencies (currently none needed)
npm install

# Find next priority assertion to work on
npm run next

# Launch builder loop (implements specs)
./builder-loop.sh

# Launch coach loop (creates specs)
./coach-loop.sh
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

### 3. The Parser

The spec parser (`src/parser/`) reads all specs and identifies the next priority work item:

```bash
$ npm run next
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

### 4. Agent Workflows

#### Builder Loop
Automates implementation of specs:

```bash
./builder-loop.sh
```

The builder:
1. Gets next priority assertion via parser
2. Reads the assertion requirements
3. Implements the feature/fix
4. Runs tests
5. Commits changes
6. Repeats until all assertions done

#### Coach Loop
Helps create well-formed specs:

```bash
./coach-loop.sh
```

The coach:
1. Takes user feature requests
2. Asks clarifying questions
3. Creates detailed spec files
4. Ensures specs follow format
5. Commits new specs

## Commands

### Parser Commands

```bash
# Get next priority assertion
npm run next

# Get all specs (full hierarchy)
node src/parser/cli.js --all
```

### Agent Commands

```bash
# Launch coach agent (via npm)
npm run coach

# Launch builder agent (via npm)
npm run builder

# Or use the CLI directly
./bin/spekk.js coach
./bin/spekk.js builder
./bin/spekk.js          # Default: runs parser
```

### Loop Commands

```bash
# Automated builder loop (implements specs continuously)
./builder-loop.sh

# Automated coach loop (creates specs continuously)
./coach-loop.sh
```

## Directory Structure

```
spekk-cli/
├── src/                    # All implementation code
│   ├── parser/            # Spec parser
│   │   ├── index.js       # Core logic
│   │   └── cli.js         # CLI interface
│   ├── coach/             # Coach agent
│   │   └── cli.js
│   └── builder/           # Builder agent
│       └── cli.js
├── bin/
│   └── spekk.js          # Main CLI entry point
├── specs/                 # All specifications
│   ├── spec-parser/
│   ├── coach-agent/
│   ├── builder-agent/
│   └── spekk-cli/
├── builder-loop.sh       # Automated builder workflow
├── coach-loop.sh         # Automated coach workflow
└── package.json
```

## Development Workflow

### Creating New Features

1. **Write the spec** (or use coach agent):
   ```bash
   npm run coach
   ```

2. **Implement via builder loop**:
   ```bash
   ./builder-loop.sh
   ```

3. **Or implement manually**:
   ```bash
   # Check what's next
   npm run next

   # Read the assertion file
   cat specs/spec-parser/assertions/parses-frontmatter.md

   # Implement the feature
   # ... write code ...

   # Update status to done
   # Edit frontmatter: status: not_started → status: done
   ```

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

## Priority Levels

- **1** - Highest priority (critical, blocking)
- **2** - Medium priority (important, not blocking)
- **3** - Lowest priority (nice to have, future)

Keep it simple: only 3 levels. Forces clear prioritization decisions.

## Testing

The spec system has two types of tests (currently not implemented):

1. **Implementation tests** - JavaScript tests in `src/**/__tests__/`
2. **Sidecar validation tests** - Bash scripts alongside assertions (`*.test.sh`)

See `specs/spec-parser/assertions/assertions-have-tests.md` for details.

## Requirements

- **Node.js** 18+ (ES modules support)
- **Claude CLI** (for builder/coach loops)
- **Git** (for automated commits)

## Philosophy

**Spec-driven development** means:
1. Write specs first (what must be true)
2. Let AI agents implement (how to make it true)
3. Validate with tests (prove it's true)
4. Iterate continuously (keep specs updated)

The specs are the source of truth. Code is the implementation of specs. Tests prove specs are satisfied.

## More Information

- **Main spec**: `specs/spec-parser/spec-parser.md`
- **Coach agent**: `specs/coach-agent/coach-agent.prompt.md`
- **Builder agent**: `specs/builder-agent/builder-agent.prompt.md`
- **CLI spec**: `specs/spekk-cli/spekk-cli.md`

## License

MIT
