# Spekk CLI

**Spec-driven development for AI coding agents.** Write what must be true in plain markdown specs. Let agents make it true.

Spekk turns a `specs/` directory in your repository into a work queue for AI agents:

- A **spec** is a markdown file that states what must be true, split into small, testable **assertions**.
- `spekk next` parses the specs and returns the next ready assertion. It follows dependencies and branches.
- The **coach** agent turns feature requests and meeting notes into well-formed specs.
- The **builder** agent takes assertions and implements them until they are true.
- The **observer** agent finds drift between the specs and the code.
- Spekk works inside [your coding assistant](#use-spekk-from-your-coding-assistant) (Claude Code, Copilot, Cursor, OpenCode, Codex) or on its own from the terminal.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh
```

Or with Go: `go install github.com/spekk-ai/spekk-cli/cmd/spekk@latest`. Or download a binary from the [releases page](https://github.com/spekk-ai/spekk-cli/releases/latest).

> **Platforms:** Spekk supports macOS and Linux. Windows binaries are published but not tested. Use [WSL](https://learn.microsoft.com/en-us/windows/wsl/) on Windows.

## Quick Start

The workflow is three commands:

```bash
cd your-project
spekk init       # 1. Create the specs/ directory
spekk coach      # 2. Draft specs with the coach agent
spekk builder    # 3. Implement the next ready assertion
```

**With Claude Code:** `spekk coach` and `spekk builder` launch Claude Code with the agent loaded. The three commands above are all you need.

**With another assistant (Copilot, Cursor, OpenCode, Codex):** register the agents and run the same workflow from inside it:

```bash
spekk install --target cursor    # or: claude-code, copilot, opencode, codex
```

Then ask **spekk-coach** to draft specs and **spekk-builder** to implement them, in your normal session. See [Use Spekk from Your Coding Assistant](#use-spekk-from-your-coding-assistant).

These commands work from any terminal:

```bash
spekk next       # Print the next ready assertion
spekk status     # Overview of every spec and assertion
spekk validate   # Check the spec tree, exit 1 on a fault
spekk show -w    # Spec explorer in the browser, with live reload
```

**Documentation:** [Getting started](./docs/getting-started.md) · [Concepts](./docs/concepts.md) · [CLI reference](./docs/cli-reference.md) · [Configuration](./docs/configuration.md) · [Skills](./docs/coach-skills.md) · [Validation in CI](./docs/ci.md)

## Use Spekk from Your Coding Assistant

`spekk coach` and `spekk builder` launch Claude Code, but the agents work from any coding assistant. `spekk install` writes them into your tool:

```bash
spekk install --target claude-code   # ~/.claude/agents/ and ~/.claude/skills/
spekk install --target copilot       # ~/.copilot/agents/ (VS Code)
spekk install --target cursor        # ~/.cursor/agents/ and ~/.cursor/commands/
spekk install --target opencode      # ~/.config/opencode/agents/ and skills/
spekk install --target codex         # ~/.codex/prompts/
```

The observer is installed as an agent. The coach, the builder, and the `spekk-dev-loop` are installed as skills. The agent and role files are thin: each one runs `spekk prompt <agent>` at session start, so the installed agents always match your binary. An update to spekk updates every tool at the same time. Add `--project` to install into the current repository instead of your home directory. See [`spekk install`](./docs/cli-reference.md#spekk-install-target-tool) for the paths per tool.

The agents engage only in a project that has a `specs/` directory. In every other project they stand down.

### Any other tool

When your assistant is not in the list, you do not need an installer. A tool that can run a shell command, or accept pasted text, can use spekk:

```bash
spekk prompt coach      # Print the full coach prompt
spekk prompt builder    # Print the full builder prompt
spekk skill list coach  # List the coach skills
```

Wire `spekk prompt <agent>` into your tool's custom-agent, rules, or system-prompt mechanism. When your tool reads `AGENTS.md`, one line is enough: "For spec-driven development, run `spekk prompt coach` (or `builder`) and follow those instructions."

## Installing Skills

The agents (coach, builder, observer) load skills from three places, in order: the project (`.spekk/skills/<agent>/`), your user directory (`~/.config/spekk/skills/<agent>/`), and the skills built into the binary. `spekk install <agent> <skill>` fetches a skill from the registry, [`github.com/spekk-ai/spekk-skills`](https://github.com/spekk-ai/spekk-skills), and writes it into one of the first two.

```bash
# Into the current project (.spekk/skills/<agent>/)
spekk install coach meeting-notes

# Into your user directory, for every project (~/.config/spekk/skills/<agent>/)
spekk install coach meeting-notes --global

# From a URL instead of the registry
spekk install coach my-skill --source https://example.com/skills/my-skill.md

# Overwrite a skill file that exists
spekk install coach meeting-notes --force

# Remove a skill. Pass --local or --global to match where it is
spekk uninstall coach meeting-notes

# List every skill an agent can use, with its source
spekk skills list coach
```

`spekk install` refuses to overwrite a skill file that exists unless you pass `--force`.

### Self-Hosted Mirrors

To point `spekk install` at a fork or an internal mirror of the registry, set these variables before you run it:

- `SPEKK_SKILLS_RAW_BASE`: base URL for the raw skill files (default: `https://raw.githubusercontent.com/spekk-ai/spekk-skills/main`)
- `SPEKK_SKILLS_API_BASE`: base URL for the directory listing that `spekk install --list <agent>` reads (default: `https://api.github.com/repos/spekk-ai/spekk-skills/contents`)

```bash
export SPEKK_SKILLS_RAW_BASE=https://raw.githubusercontent.com/my-org/internal-skills/main
export SPEKK_SKILLS_API_BASE=https://api.github.com/repos/my-org/internal-skills/contents
spekk install coach my-skill
```

## Customizing Agent Prompts

Spekk layers the agent prompts (coach, builder, observer), so you can change an agent's behavior at two levels without a change to the binary.

### Extend or override

The file name says what the file does:

- `<agent>.prompt.md` **extends** the base prompt. Spekk appends your content after the built-in prompt, so you add rules without a loss of the defaults.
- `<agent>.prompt.override.md` **overrides** the base prompt. Your content replaces the built-in prompt. Extend files still apply after an override.

### Where the files go

| Location | Scope | Example |
|----------|-------|---------|
| `~/.config/spekk/` | Global, every project | `~/.config/spekk/builder.prompt.md` |
| `.spekk/` at the project root | Local, this project | `.spekk/coach.prompt.md` |

A local file wins over a global file. When a local override and a global override both exist, the local override wins. The global directory follows the XDG rule: `$XDG_CONFIG_HOME/spekk` when `XDG_CONFIG_HOME` is set, `~/.config/spekk` otherwise.

### Example: extend the builder for every project

Create `~/.config/spekk/builder.prompt.md`:

```markdown
## Company Standards

- Use TypeScript strict mode
- All functions must have JSDoc comments
- No console.log in production code
```

Spekk appends this to the base builder prompt in every project.

### Example: override the coach for one project

Create `.spekk/coach.prompt.override.md`:

```markdown
# Custom Coach Agent

You are a coach agent for a Django/HTMX project.
When creating specs, follow Django conventions and
reference the project's existing app structure.
```

This replaces the base coach prompt in this project. Extend files (`~/.config/spekk/coach.prompt.md` and `.spekk/coach.prompt.md`) still apply after it.

### Version control

Commit `.spekk/` when the team shares the same customizations. Add it to `.gitignore` when each person keeps their own. Both work.

## Customizing Agent Skills

Spekk finds agent skills in a layered set of directories, the same way it finds prompts. The three agents (coach, builder, observer) use the same layout, the same resolution order, and the same command form.

### Resolution order

| Layer | Path | Purpose |
|-------|------|---------|
| Local | `.spekk/skills/<agent>/*.md` | Skills for this project |
| Global | `~/.config/spekk/skills/<agent>/*.md` | Your skills, for every project |
| Package | Built into the binary | The default skills |

The first match wins. A local skill shadows a global skill with the same name, and a global skill shadows a package skill.

### Invocation

A skill in one of those directories becomes the first positional argument:

```bash
spekk coach my-skill
spekk builder my-skill
spekk observer my-skill
```

Spekk inlines the skill's markdown into the agent's activation message. A new skill needs no code change.

### Example: an observer skill for one project

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

The coach (`.spekk/skills/coach/`) and the builder (`.spekk/skills/builder/`) work the same way.

### Help lists the skills

`spekk <agent> --help` lists every skill it can find, local, global, and package, in one list.

## How It Works

### 1. Spec structure

Specs live in the `specs/` directory:

```
specs/
├── spec-parser/
│   ├── spec-parser.md           # The spec
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

### 2. Spec format

Each spec and each assertion is a markdown file with YAML frontmatter:

```markdown
---
id: my-feature
created: 2026-01-21T12:00:00Z
priority: 1                   # 1 is highest, 3 is lowest
status: not_started           # not_started | in_progress | done | draft | failed
---

# Feature Name

## What Must Be True

Describe what must exist or work for this to be complete.

## Success Criteria

- Specific, testable criteria
- Clear validation steps
```

An assertion carries the same fields, plus `parent`: the id of its spec. `status` is optional and defaults to `not_started`.

A parent spec's status is computed from its assertions. Draft assertions do not count. Any `failed` assertion makes the spec `failed`. All `done` makes it `done`. A spec with no assertions, or with only drafts, is `not_started`. Every other mix is `in_progress`. Do not write a `status` on a spec, except the literal `draft`, which hides the spec and its assertions from the queue.

### Optional fields

`depends-on`: the id of one assertion that must be `done` first.

- It makes a chain: A, then B, then C.
- The parser checks that the id exists, and refuses a cycle.
- `spekk next` skips an assertion whose dependency is not `done`.

`branch`: the git branch the assertion is on.

- The default is `main`.
- `spekk next` shows only the current branch by default.
- `spekk next --all-branches` shows every branch.

```yaml
---
id: feature-x
parent: my-feature
depends-on: prerequisite-y   # prerequisite-y must be done first
branch: feature/my-feature   # This assertion is on a feature branch
---
```

### 3. The parser

`spekk next` reads every spec and prints the next assertion to work on:

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

**Selection rules:**

1. Skip `done` and `draft` assertions, and every assertion whose spec is `draft`.
2. Keep the current branch only, unless `--all-branches`.
3. Skip an assertion whose `depends-on` target is not `done`.
4. Skip an `in_progress` assertion that a builder holds with a fresh lock.
5. A lower priority number wins. On a tie, the oldest `created` wins.

**Filters:**

```bash
spekk next --spec auth              # Next assertion in one spec
spekk next --assertion login-button # One assertion, whatever its status
spekk next --all                    # The full spec hierarchy
```

### 4. Agent workflows

#### Builder

```bash
spekk builder                # Build assertions in a loop (default)
spekk builder --once         # Build one assertion, then exit
spekk builder --dry-run      # Print the next assertion, build nothing
spekk builder --spec auth    # Only assertions in one spec
spekk builder --assertion login-button  # One assertion, whatever its status
spekk builder --confirm      # Ask before each build
spekk builder --interactive  # Work with the builder in one session
```

Each build: the builder gets the next assertion from `spekk next`, reads it, writes tests where they apply, implements the change, runs the tests, runs `spekk validate`, and commits. It repeats until every assertion is done, or exits after one build with `--once`.

#### Coach

```bash
spekk coach                    # Interactive spec creation
spekk coach meeting            # Turn a meeting transcript into specs, todos, and context
spekk coach meeting notes.txt  # The same, from a file
spekk coach coordinate         # Plan dependencies and branches
```

#### Observer

```bash
spekk observer                 # Run one scan, file one observation
spekk observer install-cron    # Run it once a day from crontab
```

The observer finds a change to the code that the specs do not record, and a change to the specs that the code does not implement. Each run files one observation on an `observer/<slug>` branch and stops.

## Commands

```bash
spekk              # Same as spekk next
spekk init         # Set up a project (creates specs/)
spekk next         # Print the next ready assertion
spekk list         # List assertions as a table, JSON, TSV, or CSV
spekk status       # Overview of every spec and assertion
spekk validate     # Check the spec tree, exit 1 on a fault
spekk index        # Build the SQLite index by hand
spekk query        # Run a SELECT against the index
spekk show         # Spec explorer in the browser
spekk show -w      # The same, with live reload
spekk coach        # Launch the coach agent
spekk builder      # Launch the builder agent
spekk observer     # Launch the observer agent
spekk loop         # Run an agent in a loop that commits after each session
spekk install      # Install the agents into a coding assistant, or a skill from the registry
spekk uninstall    # Remove an installed skill
spekk skills       # List the skills an agent can use
spekk skill        # List and print agent skills
spekk prompt       # Print an agent's resolved prompt
spekk serve        # WebSocket server for the browser extension
spekk sandbox      # Manage sandboxes for remote agents
spekk conversation # Ask for a conversation, from inside a sandbox
spekk update       # Update to the latest release
spekk version      # Print the version
spekk help         # Print the command list
```

The [CLI reference](./docs/cli-reference.md) has every flag.

### Sandbox (remote agents)

`spekk sandbox` creates a DigitalOcean droplet, or registers a machine you already have, and installs a generic Claude Code agent on it. The agent connects **out** to a control host over WebSocket. It knows nothing about specs. The control host decides what to send, and the agent runs it. See [Sandbox Architecture](./docs/advanced/sandbox-architecture.md) for the connection model and the message protocol, and [Configuration](./docs/configuration.md#sandbox-provisioning) for the environment variables and the auth modes.

## Contributing

[CONTRIBUTING.md](./CONTRIBUTING.md) covers the development setup, the project structure, the tests, and the release process. This repository is built with spekk. `spekk next` shows what is ready to work on.

## Requirements

- **Git**, for branch-aware parsing and the commits the agents make
- **Claude Code CLI**, for the `spekk coach`, `spekk builder`, and `spekk observer` launchers only. You do not need it when you use the agents [from another assistant](#use-spekk-from-your-coding-assistant)
- **Go** 1.25 or later, only to build from source

## Philosophy

Spec-driven development means:

1. Write the specs first: what must be true.
2. Let AI agents implement them: how to make it true.
3. Validate with tests: prove it is true.
4. Iterate: keep the specs current.

The specs are the source of truth. The code implements the specs. The tests prove the specs are satisfied.

## License

[Apache 2.0](./LICENSE)
