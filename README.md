# Spekk CLI

**Spec-driven development for AI coding agents.** Describe what must be true in plain-markdown specs; let agents make it true.

Spekk turns a `specs/` directory in your repo into a work queue for AI agents:

- **Specs** are markdown files stating what must be true, broken into small, testable **assertions**
- `spekk next` parses them and returns the next ready assertion — dependency- and branch-aware
- The **coach** agent turns messy feature requests and meeting notes into well-formed specs
- The **builder** agent picks up assertions and implements them until they're true
- Works inside [your existing coding assistant](#use-spekk-from-your-coding-assistant) — Claude Code, Copilot, Cursor, OpenCode, Codex — or standalone from the terminal

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh
```

Or with Go: `go install github.com/spekk-ai/spekk-cli/cmd/spekk@latest` — or download a binary (including Windows) from [releases](https://github.com/spekk-ai/spekk-cli/releases/latest).

## Quick Start

The workflow is three commands:

```bash
cd your-project
spekk init       # 1. create the specs/ directory
spekk coach      # 2. draft specs with the coach agent
spekk builder    # 3. implement the next ready assertion
```

**Using Claude Code?** `spekk coach` and `spekk builder` launch it directly with the agent loaded — the commands above are all you need.

**Using another assistant (Copilot, Cursor, OpenCode, Codex)?** Register the agents as subagents and run the same workflow from inside it:

```bash
spekk install --target cursor    # or: claude-code, copilot, opencode, codex
```

Then ask **spekk-coach** to draft specs and **spekk-builder** to implement them, right in your normal session. See [Use Spekk from Your Coding Assistant](#use-spekk-from-your-coding-assistant) for details.

Either way, these work from any terminal:

```bash
spekk next       # print the next ready assertion (dependency-aware)
spekk status     # overview of all specs and assertions
spekk show -w    # interactive spec explorer with live reload
```

**Full documentation:** [Getting started](./docs/getting-started.md) · [Concepts](./docs/concepts.md) · [CLI reference](./docs/cli-reference.md) · [Configuration](./docs/configuration.md) · [Coach skills](./docs/coach-skills.md)

## Use Spekk from Your Coding Assistant

`spekk coach` and `spekk builder` launch Claude Code, but the agents work from any coding assistant. `spekk install` registers them as subagents in your preferred tool:

```bash
spekk install --target claude-code   # ~/.claude/agents/
spekk install --target copilot      # ~/.copilot/agents/ (VS Code)
spekk install --target cursor       # ~/.cursor/agents/
spekk install --target opencode     # ~/.config/opencode/agents/
spekk install --target codex        # ~/.codex/prompts/
```

This writes thin shims that fetch their instructions from the `spekk` binary at session start (via `spekk prompt <agent>`), so installed agents always match your binary version — updating spekk updates every tool at once. Add `--project` to install into the current repo instead of globally.

The agents are designed to be unintrusive: they only engage in projects that have a `specs/` directory, and stay dormant everywhere else.

### Any other tool

If your assistant isn't listed, you don't need an installer — any tool that can run a shell command (or accept pasted text) can use spekk:

```bash
spekk prompt coach      # print the full coach prompt
spekk prompt builder    # print the full builder prompt
spekk skill list coach  # list available coach skills
```

Wire `spekk prompt <agent>` into your tool's custom-agent, rules, or system-prompt mechanism. If your tool reads `AGENTS.md`, a single line like *"For spec-driven development tasks, run `spekk prompt coach` (or `builder`) and follow those instructions"* is enough.

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
