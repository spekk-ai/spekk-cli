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

Or with Go: `go install github.com/spekk-ai/spekk-cli/cmd/spekk@latest` — or download a binary from [releases](https://github.com/spekk-ai/spekk-cli/releases/latest).

> **Platform support:** Spekk supports **macOS and Linux**. Windows is not officially supported — Windows binaries are published on a best-effort basis and untested; use [WSL](https://learn.microsoft.com/en-us/windows/wsl/) for the supported experience.

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

## Installing Skills

Agents (coach, builder, observer) load skills from a layered set of directories: project-local (`<cwd>/.spekk/skills/<agent>/`), global (`~/.config/spekk/skills/<agent>/`), and the embedded defaults baked into the binary. The `spekk install` command fetches skills from the official registry — [`github.com/spekk-ai/spekk-skills`](https://github.com/spekk-ai/spekk-skills) — and drops them into one of those directories.

### Common Invocations

```bash
# Install from the registry into the current project (.spekk/skills/<agent>/)
spekk install coach meeting-notes

# Install globally so every project on your machine can see it (~/.config/spekk/skills/<agent>/)
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
| `~/.config/spekk/` | Global -- applies to all your projects | `~/.config/spekk/builder.prompt.md` |
| `.spekk/` (project root) | Local -- applies to this project only | `.spekk/coach.prompt.md` |

Local files take precedence over global files. If both a local override and a global override exist, the local override wins.

### Example: Extending the Builder Prompt Globally

Create `~/.config/spekk/builder.prompt.md`:

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

This completely replaces the base coach prompt for this project. Any extend files (`~/.config/spekk/coach.prompt.md` or `.spekk/coach.prompt.md`) are still appended after the override.

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

Finds spec-code drift and detects when code changes but specs don't (or vice versa):

```bash
spekk observer
```

## Commands

### Core Commands

```bash
spekk              # Default: runs parser (equivalent to `spekk next`)
spekk init         # Set up a project (creates specs/)
spekk next         # Get next priority assertion
spekk show         # Launch interactive web spec explorer
spekk show -w      # Watch mode with live reload
spekk status       # Comprehensive overview of all specs/assertions
spekk coach        # Launch Coach Agent
spekk builder      # Launch Builder Agent
spekk observer     # Launch Observer Agent (finds drift)
spekk install      # Install agents into a coding assistant (--target)
spekk prompt       # Print an agent's resolved prompt
spekk skill        # List and print agent skills (list, show)
spekk serve        # Start WebSocket server for browser extension
spekk sandbox      # Manage cloud sandbox environments (remote agents)
spekk loop         # Run orchestration workflows
spekk update       # Self-update to the latest release
spekk version      # Print the current version
spekk help         # Show help message
```

### Sandbox (Remote Agents)

`spekk sandbox` provisions cloud VMs running a generic Claude Code agent. The agent connects **out** to a control host over WebSocket — it is not spec-aware and knows nothing about the spekk workflow. The control host decides what to send; the agent runs it. See [Sandbox Architecture](./docs/advanced/sandbox-architecture.md) for the connection model, message protocol, and worker pool details.

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

## Contributing

Development setup, project structure, testing, and the release process are documented in [CONTRIBUTING.md](./CONTRIBUTING.md). This repo is built with spekk itself — `spekk next` shows what's ready to work on.

## Requirements

- **Git** (for branch-aware parsing and automated commits)
- **Claude CLI** — only for the standalone `spekk coach` / `spekk builder` / `spekk observer` launchers; not needed when using the agents [from another assistant](#use-spekk-from-your-coding-assistant)
- **Go** 1.23+ — only for building from source

## Philosophy

**Spec-driven development** means:
1. Write specs first (what must be true)
2. Let AI agents implement (how to make it true)
3. Validate with tests (prove it's true)
4. Iterate continuously (keep specs updated)

The specs are the source of truth. Code is the implementation of specs. Tests prove specs are satisfied.

## License

[Apache 2.0](./LICENSE)
