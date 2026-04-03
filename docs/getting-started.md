---
icon: lucide/rocket
---

# Getting Started

Get up and running with Spekk CLI in under five minutes.

## Prerequisites

- **Node.js** 18+ (ES modules support)
- **Claude CLI** (for builder/coach/observer agents)
- **Git** (for automated commits)

## Installation

### 1. Get a GemFury token

Ask a team lead for a GemFury **read** (or full access) token for the `thinknimble` org.

### 2. Add the token to your shell

```bash
# Add to ~/.zshrc or ~/.bashrc
export GEMFURY_SPEKK_TOKEN=your_token_here
```

Reload your shell:

```bash
source ~/.zshrc  # or ~/.bashrc
```

### 3. Configure npm

Add to your global `~/.npmrc`:

```ini
@spekk:registry=https://npm.fury.io/thinknimble/
//npm.fury.io/thinknimble/:_authToken=${GEMFURY_SPEKK_TOKEN}
```

Or run:

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

---

## Your first spec

### Create a spec with the coach

```bash
spekk coach
```

Tell the coach what you want to build:

```
> "I need to add user authentication to my app"
```

The coach guides you through creating a well-formed specification with clear success criteria, atomic assertions, and priority ordering.

### View your specs

```bash
# What's next?
spekk next

# Full overview
spekk status

# Interactive web explorer
spekk show
```

### Build it

```bash
# Build the next priority assertion
spekk builder --once

# Or loop through all assertions
spekk builder
```

---

## Directory structure

After creating specs, your project will have:

```
your-project/
├── specs/
│   ├── authentication/
│   │   ├── authentication.md        # Parent spec
│   │   └── assertions/
│   │       ├── password-hashing.md  # Atomic assertion
│   │       └── session-tokens.md    # Atomic assertion
│   └── another-feature/
│       ├── another-feature.md
│       └── assertions/
│           └── ...
├── TODOS.md          # Action items (from meetings)
└── CONTEXT.md        # Architecture decisions
```

---

## Common workflows

### Creating a new feature

```bash
# 1. Define the spec
spekk coach

# 2. Coordinate dependencies
spekk coach coordinate

# 3. Create feature branch
git checkout -b feature/authentication

# 4. Build it
spekk builder --once
```

### Processing meeting notes

```bash
# Process a transcript
spekk coach meeting notes.txt

# Review extracted todos
cat TODOS.md

# Review new specs
spekk next --all
```

### Planning a sprint

```bash
# See what's ready
spekk status

# Coordinate work across branches
spekk coach coordinate

# Start building
spekk builder
```

---

## Next steps

- [Concepts](concepts.md) -- Understand specs, assertions, and priorities
- [CLI Reference](cli-reference.md) -- Complete command documentation
- [Coach Skills](coach-skills.md) -- Meeting notes, coordination, and more
- [Configuration](configuration.md) -- Customize agent prompts
