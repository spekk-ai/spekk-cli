# Getting Started with Spekk

Spekk is a spec-driven development CLI that helps you write better specifications and automate development workflows.

## Installation

```bash
npm install -g @spekk/cli
```

## Quick Start

### 1. Create Your First Spec

```bash
# Launch the coach to help you write specs
spekk coach

# Tell it what you want to build
> "I need to add user authentication to my app"
```

The coach will guide you through creating a well-formed specification with:
- Clear success criteria
- Atomic assertions
- Priority ordering

### 2. View Your Specs

```bash
# See what needs to be built next
spekk next

# View all specs
spekk next --all

# Get comprehensive overview
spekk status
```

### 3. Build Your Specs

```bash
# Build one assertion at a time
spekk builder --once

# Or loop through all assertions
spekk builder
```

## Key Concepts

### Specs vs Assertions

- **Specs** define what you're building (e.g., "User Authentication")
- **Assertions** are atomic units of work (e.g., "Password is hashed with bcrypt")

### Spec-Driven Workflow

1. **Write specs** - Define what must be true (not how to build it)
2. **Prioritize** - Order by importance (1 = highest, 3 = lowest)
3. **Build** - Implement assertions using the builder
4. **Validate** - Tests prove specs are satisfied
5. **Iterate** - Keep specs updated as you learn

## Coach Skills

The coach has specialized skills for common tasks:

### Meeting Notes to Specs

Process meeting transcripts and extract todos, specs, and context:

```bash
# Interactive mode
spekk coach meeting

# Or provide transcript file
spekk coach meeting notes.txt
```

### Work Coordination

Analyze assertions and create dependency-aware work plan:

```bash
spekk coach coordinate
```

Creates:
- Dependency graph (what builds in what order)
- Branch assignments (what can be built in parallel)
- YAML frontmatter updates (depends-on, branch fields)

### Business Model Validation

Assess startup/business ideas through structured questions:

```bash
spekk coach
> "validate my business model"
```

## Directory Structure

```
your-project/
├── specs/
│   ├── authentication/
│   │   ├── authentication.md        # Parent spec
│   │   └── assertions/
│   │       ├── password-hashing.md
│   │       └── session-tokens.md
│   └── parser/
│       ├── parser.md
│       └── assertions/
│           └── parses-yaml.md
```

## Next Steps

- [CLI Reference](./CLI-REFERENCE.md) - Complete command documentation
- [Coach Skills Guide](./COACH-SKILLS.md) - Detailed skill workflows
- [Spec Format Guide](../README.md#spec-format) - How to write specs
