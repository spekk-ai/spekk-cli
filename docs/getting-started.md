---
icon: lucide/rocket
---

# Getting Started

Get up and running with Spekk CLI in under five minutes.

## Prerequisites

- **Claude CLI** (for builder/coach/observer agents)
- **Git** (for automated commits)
- **Go** 1.23+ (only if building from source)

## Installation

### Install script (recommended)

Detects your platform, installs to user-owned `~/.local/bin` (no sudo needed — for installs or updates), and warns if the directory isn't on your `PATH`:

```bash
curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh
```

See the [install guide](install.md) for manual downloads and options.

### From Source

```bash
go install github.com/spekk-ai/spekk-cli/cmd/spekk@latest
```

### Verify

```bash
spekk --help
```

### Updating

Download the latest release binary again using the same curl command above, or rebuild from source with `go install`.

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
