---
icon: lucide/square-terminal
---

# CLI Reference

Complete reference for all Spekk CLI commands.

---

## `spekk next`

Show the next priority assertion to work on.

```bash
spekk next                          # Next on current branch
spekk next --all-branches           # Next across all branches
spekk next --spec authentication    # Next in specific spec
spekk next --assertion password-hashing  # Details for specific assertion
spekk next --all                    # Full spec hierarchy (JSON)
spekk next --raw                    # Raw JSON for downstream processing
spekk next --specs-dir ./my-specs   # Custom specs directory
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--all` | | Show all assertions |
| `--all-branches` | | Include assertions from all branches |
| `--spec <name>` | `-s` | Filter by spec name |
| `--assertion <id>` | | Filter by assertion ID |
| `--raw` | | Output raw JSON for downstream processing |
| `--specs-dir <path>` | | Path to specs directory (default: `./specs`) |

**Selection logic:**

1. Filters to current git branch (unless `--all-branches`)
2. Removes `done` and `draft` assertions
3. Skips assertions with unfinished dependencies
4. Sorts by priority (1 > 2 > 3)
5. Breaks ties by `created` timestamp (older first)

**Output fields:**

| Field | Description |
|-------|-------------|
| `type` | `"assertion"` or `"spec"` |
| `id` | Assertion identifier |
| `parent` | Parent spec ID |
| `file` | Path to markdown file |
| `priority` | 1 (high), 2 (medium), 3 (low) |
| `status` | Current state |
| `branch` | Assigned branch |
| `dependsOn` | Prerequisite assertion ID |

---

## `spekk builder`

Launch the builder agent to implement assertions.

```bash
spekk builder                       # Loop through all assertions
spekk builder --once                # Build one, then exit
spekk builder --dry-run             # Preview without executing
spekk builder --spec authentication # Only this spec's assertions
spekk builder --assertion login     # Build specific assertion
spekk builder --confirm             # Ask before each build
spekk builder --interactive         # Collaborate with the builder
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| *(none)* | | Loop through all assertions continuously |
| `--once` | | Build one assertion then exit |
| `--dry-run` | `-d` | Preview what would be built |
| `--spec <id>` | `-s` | Only assertions in this spec |
| `--assertion <id>` | | Specific assertion (even if done) |
| `--confirm` | `-c` | Confirm before each build |
| `--interactive` | `-i` | Interactive/collaborative mode |

**How it works:**

1. Gets next priority assertion via parser
2. Reads the assertion requirements
3. Writes tests (when applicable)
4. Implements the feature/fix
5. Runs tests to validate
6. Commits changes
7. Repeats (unless `--once`)

??? info "Lean testing philosophy"

    The builder follows a lean testing approach:

    - Tests behavior, not implementation details
    - One test per meaningful behavior
    - Deletes redundant or low-value tests
    - No tests for trivial code
    - Prefers integration tests over unit when appropriate

---

## `spekk coach`

Launch the coach agent for interactive spec creation.

```bash
spekk coach                          # Interactive mode
spekk coach meeting                  # Process meeting transcript
spekk coach meeting notes.txt        # Process transcript file
spekk coach coordinate               # Plan dependencies
```

### `spekk coach` (interactive)

Helps you write well-formed specifications:

- Asks clarifying questions about requirements
- Creates spec and assertion files
- Ensures correct format and structure
- Commits changes to git

### `spekk coach meeting [file]`

Process meeting transcripts into structured outputs:

- **Todos** → appended to `TODOS.md`
- **Specs** → proper spec files in `specs/`
- **Context** → architectural decisions appended to `CONTEXT.md`

```bash
# Interactive (paste transcript)
spekk coach meeting

# From file
spekk coach meeting standup-notes.txt
```

### `spekk coach coordinate`

Create a dependency-aware work plan:

1. Reads all `draft` and `not_started` assertions
2. Analyzes dependencies
3. Groups related work into feature branches
4. Shows dependency tree for approval
5. Updates YAML frontmatter (`depends-on`, `branch`)
6. Validates with parser
7. Commits changes

---

## `spekk status`

Get a comprehensive overview of all specs.

```bash
spekk status
```

Shows total specs/assertions, status breakdown, completion percentage, and specs grouped by status.

---

## `spekk show`

Launch the interactive web-based spec explorer.

```bash
spekk show             # Open spec explorer
spekk show --watch     # Live reload on file changes
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--watch` | `-w` | Enable live reload via SSE when spec files change |

Opens a browser with:

- Expandable spec/assertion hierarchy
- Status and priority badges
- Dependency metro map visualization
- Drag-to-pan, click-to-navigate
- Completed specs hidden by default (toggle to show)
- Searchbar filtering by name, status, or priority

---

## `spekk observer`

Monitor spec-code drift.

```bash
spekk observer                    # Default monitoring
spekk observer --interval 30     # Check every 30 seconds
spekk observer --quiet            # Minimal output
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--interval <seconds>` | Preferred scan interval in seconds (positive integer) |
| `--quiet` | Minimal output mode |

Detects when code changes but specs don't update (or vice versa). Helps keep specs and implementation synchronized.

---

## `spekk loop`

Run orchestration workflows.

```bash
spekk loop builder    # Automated builder loop
spekk loop coach      # Interactive coach loop
```

### `spekk loop builder`

Runs the automated builder loop: gets next assertion, implements, commits, repeats.

### `spekk loop coach`

Runs the interactive coach loop: create specs, commit, repeat.

---

## `spekk serve`

Start a WebSocket server for browser extension integration.

```bash
spekk serve                        # Default (localhost:3118)
spekk serve --port 8080            # Custom port
spekk serve --host 0.0.0.0        # Bind to all interfaces
spekk serve --verbose              # Debug logging
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--port` | `-p` | Port to listen on (default: `3118`) |
| `--host` | | Host to bind to (default: `localhost`) |
| `--verbose` | `-v` | Enable debug logging for WebSocket messages |

Enables real-time communication between the CLI and the Spekk web UI.

---

## `spekk sandbox`

Manage cloud sandbox environments on DigitalOcean.

### `spekk sandbox create`

Provision a new sandbox droplet.

```bash
spekk sandbox create --name my-sandbox
spekk sandbox create --name my-sandbox --region sfo3 --size s-4vcpu-8gb
spekk sandbox create --name my-sandbox --project "My Project" --vpc <uuid>
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--name` | Sandbox name (**required**) |
| `--region` | DigitalOcean region (default: `nyc1`) |
| `--size` | Droplet size slug (default: `s-2vcpu-4gb`) |
| `--project` | Assign to a DigitalOcean project by name or UUID |
| `--vpc` | Place droplet in a specific VPC |

Creates a droplet with cloud-init, waits for SSH readiness, injects credentials, configures git/gh, and deploys the agent client.

### `spekk sandbox list`

Display all tracked sandboxes in a table format.

```bash
spekk sandbox list
```

### `spekk sandbox status <name>`

Show detailed status of a sandbox including live DO API and SSH connectivity checks.

```bash
spekk sandbox status my-sandbox
```

### `spekk sandbox ssh <name> [ssh-flags...]`

Open an interactive SSH session to a sandbox. Additional arguments are passed through to `ssh`.

```bash
spekk sandbox ssh my-sandbox
spekk sandbox ssh my-sandbox -L 8080:localhost:8080
```

### `spekk sandbox destroy <name>`

Tear down a sandbox droplet and remove local metadata.

```bash
spekk sandbox destroy my-sandbox
spekk sandbox destroy my-sandbox --force   # Skip confirmation
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--force` | `-f` | Skip confirmation prompt |

### `spekk sandbox deploy <name>`

Download and deploy the latest agent binary to an existing sandbox.

```bash
spekk sandbox deploy my-sandbox
```

---

## `spekk prompt`

Print an agent's resolved prompt to stdout.

```bash
spekk prompt coach       # Print the coach prompt
spekk prompt builder     # Print the builder prompt
spekk prompt observer    # Print the observer prompt
```

The prompt is resolved through the standard layers (`.spekk/` overrides and extensions, then the embedded base), so the output is exactly what `spekk <agent>` would launch with. Useful for piping into other tools, or for agents installed via `spekk install`, which fetch their instructions this way at session start.

---

## `spekk skill`

Discover and print agent skills.

```bash
spekk skill list coach                     # List skills and their source
spekk skill show coach coordinator-skill   # Print a skill's content
```

Skills resolve through layers: `.spekk/skills/<agent>/` (project), then `~/.spekk/skills/<agent>/` (user), then the skills built into the binary.

---

## `spekk install`

Install spekk agents (coach, builder, observer) into a coding assistant as subagents.

```bash
spekk install --target claude-code   # ~/.claude/agents/
spekk install --target copilot       # ~/.copilot/agents/ (VS Code)
spekk install --target cursor        # ~/.cursor/agents/
spekk install --target opencode      # ~/.config/opencode/agents/
spekk install --target codex         # ~/.codex/prompts/ (global only)
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--target <tool>` | Host tool to install into (required) |
| `--project` | Install into the current project instead of globally |

Installs thin shims that fetch their full instructions from the binary at session start via `spekk prompt <agent>`, so they never go stale — updating spekk updates every installed agent. For tools not listed, wire `spekk prompt <agent>` into the tool's custom-agent or rules mechanism directly.

---

## `spekk help`

Show help message.

```bash
spekk help
spekk <command> --help    # Help for specific command
```

---

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Error (check message) |

---
