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

## Installation

### From GitHub Releases (recommended)

Download the latest binary for your platform from [GitHub Releases](https://github.com/spekk-ai/spekk-cli/releases):

```bash
# macOS (Apple Silicon)
curl -L https://github.com/spekk-ai/spekk-cli/releases/latest/download/spekk-darwin-arm64 -o /usr/local/bin/spekk
chmod +x /usr/local/bin/spekk

# macOS (Intel)
curl -L https://github.com/spekk-ai/spekk-cli/releases/latest/download/spekk-darwin-amd64 -o /usr/local/bin/spekk
chmod +x /usr/local/bin/spekk

# Linux (x86_64)
curl -L https://github.com/spekk-ai/spekk-cli/releases/latest/download/spekk-linux-amd64 -o /usr/local/bin/spekk
chmod +x /usr/local/bin/spekk

# Linux (ARM64)
curl -L https://github.com/spekk-ai/spekk-cli/releases/latest/download/spekk-linux-arm64 -o /usr/local/bin/spekk
chmod +x /usr/local/bin/spekk
```

### From Source

```bash
go install github.com/spekk-ai/spekk-cli/cmd/spekk@latest
```

### Verify

```bash
spekk --help
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

# Watch mode with live reload
spekk show -w

# Get comprehensive overview of all specs
spekk status
```

## Installing Skills

Agents (coach, builder, observer) load skills from a layered set of directories: project-local (`<cwd>/.spekk/skills/<agent>/`), global (`~/.spekk/skills/<agent>/`), and the embedded defaults baked into the binary. The `spekk install` command fetches skills from the official registry — [`github.com/spekk-ai/spekk-skills`](https://github.com/spekk-ai/spekk-skills) — and drops them into one of those directories.

### Common Invocations

```bash
# Install from the registry into the current project (.spekk/skills/<agent>/)
spekk install coach meeting-notes

# Install globally so every project on your machine can see it (~/.spekk/skills/<agent>/)
spekk install coach meeting-notes --global

# Install from an arbitrary URL instead of the registry
spekk install coach my-skill --source https://example.com/skills/my-skill.md

# Uninstall (mirror of install — pick --local or --global to match where it lives)
spekk uninstall coach meeting-notes
```

To list everything available to an agent (local + global + embedded):

```bash
spekk skills list coach
```

### Overwriting Existing Skills

If a skill file already exists at the destination, `spekk install` refuses to clobber it. Pass `--force` to overwrite:

```bash
spekk install coach meeting-notes --force
```

### Self-Hosted Mirrors

The default registry is `github.com/spekk-ai/spekk-skills`. To point `spekk install` at a fork or internal mirror, set these environment variables before running the command:

- `SPEKK_SKILLS_RAW_BASE` — base URL for raw skill content (default: `https://raw.githubusercontent.com/spekk-ai/spekk-skills/main`)
- `SPEKK_SKILLS_API_BASE` — base URL for the directory-listing contents API used by `spekk install --list <agent>` (default: `https://api.github.com/repos/spekk-ai/spekk-skills/contents`)

```bash
export SPEKK_SKILLS_RAW_BASE=https://raw.githubusercontent.com/my-org/internal-skills/main
export SPEKK_SKILLS_API_BASE=https://api.github.com/repos/my-org/internal-skills/contents
spekk install coach my-skill
```

## Customizing Agent Prompts

Spekk uses a layered prompt system that lets you customize agent behavior (coach, builder, observer) at two levels without modifying the binary.

### Extend vs Override

The file naming convention determines how your customization is applied:

- **`<agent>.prompt.md`** -- **extends** the base prompt. Your content is appended after the built-in prompt, so you add rules or context without losing defaults.
- **`<agent>.prompt.override.md`** -- **overrides** the base prompt entirely. Your content replaces the built-in prompt. Extend files still layer on top of an override.

### Where to Put Customization Files

| Location | Scope | Example Path |
|----------|-------|-------------|
| `~/.spekk/` | Global -- applies to all your projects | `~/.spekk/builder.prompt.md` |
| `.spekk/` (project root) | Local -- applies to this project only | `.spekk/coach.prompt.md` |

Local files take precedence over global files. If both a local override and a global override exist, the local override wins.

### Example: Extending the Builder Prompt Globally

Create `~/.spekk/builder.prompt.md`:

```markdown
## Company Standards

- Use TypeScript strict mode
- All functions must have JSDoc comments
- No console.log in production code
```

This is appended to the base builder prompt for every project you work on.

### Example: Overriding the Coach Prompt for a Project

Create `.spekk/coach.prompt.override.md`:

```markdown
# Custom Coach Agent

You are a coach agent for a Django/HTMX project.
When creating specs, follow Django conventions and
reference the project's existing app structure.
```

This completely replaces the base coach prompt for this project. Any extend files (`~/.spekk/coach.prompt.md` or `.spekk/coach.prompt.md`) are still appended after the override.

### Version Control

The `.spekk/` directory can be committed to your repo so the whole team shares the same prompt customizations, or added to `.gitignore` if you prefer individual configuration. Choose whichever approach fits your team.

## Customizing Agent Skills

Spekk discovers agent skills from a layered set of directories, mirroring the prompt system. Skills work identically across all three agents (coach, builder, observer): same directory layout, same resolution order, same CLI invocation pattern.

### Resolution Order (first match wins)

| Layer | Path | Purpose |
|-------|------|---------|
| Local | `.spekk/skills/<agent>/*.md` | Project-specific skills |
| Global | `~/.spekk/skills/<agent>/*.md` | User's personal skills across all projects |
| Package | Ships with spekk (embedded) | Built-in skills |

A local skill shadows a global skill of the same name; a global skill shadows a package skill.

### Invocation

Any skill discovered in those directories becomes invocable as the first positional argument:

```bash
spekk coach my-skill
spekk builder my-skill
spekk observer my-skill
```

The skill's full markdown content is inlined into the agent's activation message — no code changes required to add a new skill.

### Example: Adding a Project-Specific Observer Skill

Create `.spekk/skills/observer/check-todos.md`:

```markdown
---
id: check-todos
---

# Check TODOs Skill

Scan the codebase for TODO comments older than 30 days and report them as observations.
```

Then run:

```bash
spekk observer check-todos
```

The same pattern works for coach (`.spekk/skills/coach/`) and builder (`.spekk/skills/builder/`).

### Dynamic Help

`spekk <agent> --help` lists every discovered skill, so users see local, global, and package skills together in one place.

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
│   ├── coach.prompt.md          # Agent instructions
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

- Specific, testable criteria
- Clear validation steps
```

**Parent spec status is computed** -- a parent spec's status automatically reflects its assertions' states (all done = done, any in-progress = in-progress, etc.).

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

The spec parser reads all specs and identifies the next priority work item:

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

#### Observer Agent

Monitors spec-code drift and detects when code changes but specs don't (or vice versa):

```bash
spekk observer
```

## Commands

### Core Commands

```bash
spekk              # Default: runs parser (equivalent to `spekk next`)
spekk next         # Get next priority assertion
spekk show         # Launch interactive web spec explorer
spekk show -w      # Watch mode with live reload
spekk status       # Comprehensive overview of all specs/assertions
spekk coach        # Launch Coach Agent
spekk builder      # Launch Builder Agent
spekk observer     # Launch Observer Agent (monitors drift)
spekk serve        # Start WebSocket server for browser extension
spekk sandbox      # Manage cloud sandbox environments
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

### Parser Flags

```bash
spekk next                    # Get next priority assertion
spekk next --all              # Get full spec hierarchy (JSON)
spekk next --spec <id>        # Filter to assertions in a specific spec
spekk next --assertion <id>   # Get details for specific assertion
spekk next --all-branches     # Include assertions from all branches
```

## Directory Structure

```
spekk-cli/
├── cmd/spekk/              # CLI entry point
│   └── main.go
├── internal/               # Core Go packages
│   ├── parser/            # Spec parser
│   ├── agent/             # Agent launchers (coach, builder, observer, loops)
│   ├── cli/               # Flag parsing, prompt/skill resolution
│   ├── show/              # Spec explorer web UI
│   ├── serve/             # WebSocket server for browser extension
│   ├── sandbox/           # Cloud sandbox management (DigitalOcean)
│   └── status/            # Status overview display
├── specs/                  # All specifications
│   ├── spec-parser/
│   ├── coach-agent/
│   ├── builder-agent/
│   └── ...
├── go.mod
└── go.sum
```

## Development

### Building

```bash
go build ./cmd/spekk
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/parser/

# Verbose output
go test ./... -v
```

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

   # Implement the feature, update status to done
   ```

## Requirements

- **Go** 1.23+ (for building from source)
- **Claude CLI** (for builder/coach/observer agents)
- **Git** (for automated commits)

## Philosophy

**Spec-driven development** means:
1. Write specs first (what must be true)
2. Let AI agents implement (how to make it true)
3. Validate with tests (prove it's true)
4. Iterate continuously (keep specs updated)

The specs are the source of truth. Code is the implementation of specs. Tests prove specs are satisfied.

## License

MIT
