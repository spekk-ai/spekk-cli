---
icon: lucide/square-terminal
---

# CLI Reference

Complete reference for all Spekk CLI commands.

---

## `spekk init`

Set up a project for spec-driven development.

```bash
spekk init
```

Creates a `specs/` directory (at the git root if in a repository, otherwise in the current directory) with a short README explaining the format. Does nothing if `specs/` already exists. This is the first command to run in a new project — follow it with `spekk coach` to draft your first spec.

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

## `spekk index`

Build `.spekk/index.db`, a SQLite index of the spec tree, for use by `spekk query`.

```bash
spekk index [--force] [--specs-dir <path>]
```

The Markdown files remain the source of truth; the database is a derived artifact and is added to `.gitignore` automatically. `--force` drops and recreates all tables.

You rarely need to run this by hand: the index is rebuilt automatically whenever it is stale, absent, or built against an older schema (see `spekk query` and `spekk next`). Because it is derived, an index from an older spekk version is detected via its stamped schema version and transparently rebuilt — there is no manual migration.

---

## `spekk query`

Run a read-only `SELECT` against the SQLite index and print the result.

```bash
spekk query "SELECT status, COUNT(*) FROM assertions GROUP BY status"
spekk query "SELECT id, title FROM specs WHERE status = 'draft'" --json
```

The index is refreshed automatically first, so results always reflect the current specs. Only `SELECT` (and `WITH … SELECT`) statements are permitted, and the database is opened read-only, so a query can never mutate it.

Output flags: `--json` (array of objects), `--tsv`, `--csv`. Default is a padded table.

Schema:

| Table | Columns |
|---|---|
| `specs` | `id`, `title`, `status`, `priority`, `branch`, `file` |
| `assertions` | `id`, `parent_id` (→ `specs.id`), `title`, `status`, `priority`, `branch`, `file` |
| `depends_on` | `assertion_id`, `depends_on_id` (both → `assertions.id`) |
| `frontmatter_fields` | `owner_type` (`spec` \| `assertion`), `owner_id`, `key`, `value` |

`depends_on` holds **assertion-level** edges only; spec-level relationships are not modeled as data (see the spec bodies).

### Custom frontmatter fields

Any **top-level** frontmatter key outside the known set (`id`, `parent`, `created`, `priority`, `status`, `branch`, `depends-on`, `locked-by`) is a **custom field**. `spekk validate` accepts it without warnings, and the index stores it in `frontmatter_fields` — one row per distinct value (a repeated value inserts once). Three multi-value spellings index identically: a flow sequence (`tags: [infrastructure, hipaa]`), a bare comma-separated scalar (`workflows: w1-note-and-claim, w2-claim-reimbursement`), and a YAML block list (`- item` lines).

Value rules:

- In a bare scalar or flow sequence, an **unquoted comma always splits**. A quoted region protects its commas: `note: "Hello, world"` is one value, and `[a, "b, c"]` is two.
- Block-list items are never re-split — an item keeps its commas as written.
- Comment lines, nested-map children, empty keys, and block scalars (`key: |` / `key: >`) never become custom fields. The frontmatter parser reads top-level scalars, flow sequences, and flat block lists only.

This makes per-tag reporting one query:

```bash
spekk query "SELECT f.value AS workflow, COUNT(*) AS total, SUM(a.status = 'done') AS done
  FROM assertions a
  JOIN frontmatter_fields f
    ON f.owner_type = 'assertion' AND f.owner_id = a.id AND f.key = 'workflows'
  GROUP BY f.value"
```

---

## `spekk show`

Launch the interactive web-based spec explorer.

```bash
spekk show                                          # Open spec explorer
spekk show --watch                                  # Live reload on file changes
spekk show --cross-branch                           # Merge-preview mode
spekk show --cross-branch --branch-filter 'feat/*'  # Filter branches by glob
spekk show -w --cross-branch                        # Cross-branch with live reload
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--watch` | `-w` | Enable live reload via SSE when spec files change |
| `--cross-branch` | | Read-only preview of what merging each branch into the current branch would do to the spec corpus |
| `--branch-filter` | | In cross-branch mode, only include branches matching the glob (e.g. `'feat/*'`) |

Opens a browser with:

- Expandable spec/assertion hierarchy
- Status and priority badges
- Dependency metro map visualization
- Drag-to-pan, click-to-navigate
- Completed specs hidden by default (toggle to show)
- Searchbar filtering by name, status, or priority

### Cross-branch mode

`--cross-branch` answers: *what would merging each other branch into my current branch do to the spec corpus?* It scans every local and remote-tracking branch, classifies each spec and assertion file into one of four states, and renders the results in the explorer — all without touching the working tree or index.

#### Four classification states

| Badge | State | Meaning |
|-------|-------|---------|
| `+` | Incoming addition | File exists on another branch but not on yours. Rendered as a "foreign" item with a striped background. |
| `↻` | Incoming modification | File modified on another branch, unchanged on yours — would merge cleanly. Shows assertion status drift (e.g. `not_started` → `done`). |
| `⚠` | Conflict | File modified on both branches in incompatible ways. Red border when confirmed (git >= 2.38), orange dashed border when unconfirmed. |
| `✕` | Incoming deletion | File deleted on another branch but still exists on yours. Shown with a struck-through title. |

When a spec has contributions in multiple states, the summary badge uses the highest precedence: conflict > deletion > addition > modification.

#### Branch discovery and filtering

Cross-branch mode compares against the union of local branches (`refs/heads/*`) and remote-tracking branches (`refs/remotes/*`). The current branch and symbolic refs like `origin/HEAD` are excluded. Branches with the same name in both namespaces are deduplicated (local wins).

Use `--branch-filter` to narrow the scope:

```bash
spekk show --cross-branch --branch-filter 'feat/*'    # Only feature branches
spekk show --cross-branch --branch-filter 'hotfix/*'  # Only hotfix branches
```

The filter matches against the logical branch name with any remote prefix stripped — `origin/feat/login` is matched as `feat/login`, so `feat/*` covers both local and remote-tracking branches (and `origin/*` matches nothing).

The filter uses `filepath.Match` glob semantics: `*` matches non-separator characters, `?` matches one character, `[...]` matches character classes. A malformed pattern returns an error.

#### UI elements

- **Branch banner** at the top of the view listing all compared branches
- **Branch checkbox dropdown** to filter which branches contribute — deselecting a branch hides its contributions and recalculates badges. Selection persists per project in `localStorage`
- **Inline badges** on affected specs and assertions in the tree
- **Contribution details** in the detail panel showing per-branch state and status drift
- **Foreign items** (incoming additions) appear with a diagonal-stripe background and dimmed title; they disappear when all contributing branches are deselected

#### Git version requirements

Cross-branch mode uses `git merge-tree --write-tree` (git >= 2.38) for accurate three-way conflict detection. The git version is probed once per process.

On older git, the feature **degrades gracefully** rather than failing:

- Additions, modifications, and deletions are still classified correctly
- Conflicts cannot be confirmed — both-sides-modified files are marked as *potential* conflicts with `⚠` shown in an orange dashed border
- An orange banner warns that conflict detection is unavailable

#### Read-only guarantee

All git operations are funnelled through a single allowlist chokepoint that permits only a small set of read-only subcommands (`rev-parse`, `for-each-ref`, `merge-tree`, and a few others). No checkout, merge, index mutation, or ref mutation can occur.

#### Machine-readable output — `spekk list --cross-branch`

The same classification is available as data for agents and scripts (the HTML explorer stays the human surface):

```bash
spekk list --cross-branch --json
spekk list --cross-branch --branch-filter 'feat/*' --json
spekk list --cross-branch --tsv
```

One row per changed (file, branch) pair, with columns `path`, `branch`, `state` (`incoming_add` | `incoming_mod` | `conflict` | `incoming_del`), `degraded` (a conflict that could not be confirmed on git < 2.38), and `old_status`/`new_status` (assertion status drift on a clean incoming modification, e.g. `not_started` → `done`; empty otherwise). Default output is a table; `--json` emits an array of objects (every row carries every key, and an empty set renders `[]`), and `--tsv`/`--csv` emit the same columns. This lets an observer flag, for example, an assertion `done` on `main` that a feature branch moves back to `in_progress` — before the branch merges. The same read-only guarantee applies.

#### Watch mode with cross-branch

When combined with `--watch`, cross-branch classification is cached on a git ref-state fingerprint:

- **Working-tree edits** (spec file changes) trigger a cheap re-render reusing the cached classification
- **Ref changes** (new commits, fetches, branch moves) invalidate the cache and trigger a full reclassification
- Two independent watchers run: one polling spec files and one polling git ref state
- Transient git errors are logged once and the watcher recovers automatically when git becomes available again

---

## `spekk observer`

Monitor spec-code drift.

```bash
spekk observer                    # Run one scan
spekk observer --quiet            # Minimal output
spekk observer coverage-gap       # Run with a specific skill
spekk observer prune              # Surface unused-code / consolidation candidates (recommend-only)
spekk observer consolidate        # Curate observations into a digest
spekk observer install-cron       # Schedule observer via crontab
spekk observer uninstall-cron     # Remove scheduled cron entries
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--quiet` | Minimal output mode |
| `--headless` | Run Claude in non-interactive mode (no TTY); set automatically by `install-cron` |

Detects when code changes but specs don't update (or vice versa). Helps keep specs and implementation synchronized.

**A scan is one batch, not a sweep.** A run files at most three observations and then stops, preferring what looks most load-bearing. Drift found today is still there tomorrow, so the fourth finding is the next run's first — run it again to continue. A run that stops at the cap says which areas it had not reached, so a short run is never mistaken for a clean bill of health.

Each scan closes with a quiet consolidation pass and reports only a brief summary from `observations/DIGEST.md`. When the digest is empty, the observer stays silent.

How often it runs is the schedule's business, not the run's — see `install-cron` below, or use whatever scheduler you prefer.

**Skills:** Observer supports skills with the same layered resolution as coach and builder (`.spekk/skills/observer/` > `~/.config/spekk/skills/observer/` > package > embedded). Run `spekk observer --help` to see available skills.

**Built-in skills:**

| Skill | Description |
|-------|-------------|
| `coverage-gap` | Surfaces code a spec could optionally document — a progressive-adoption aid, not a defect report (un-spec'd code is normal) |
| `prune` | Surfaces genuinely-unused code and design-level redundancy (duplication, over-abstraction, dead config) as candidates for human review — recommend-only, never deletes; a missing spec is never a signal |
| `consolidate` | Reviews all raw observations, archives stale/duplicates, rewrites `observations/DIGEST.md` with at most 5 severity-ranked items |

### `spekk observer install-cron`

Install crontab entries that run the observer on a schedule.

```bash
spekk observer install-cron                                        # Defaults: once a day
spekk observer install-cron --loop-interval 60                     # Scan every hour
spekk observer install-cron --consolidate-interval 720             # Consolidate every 12 h
spekk observer uninstall-cron                                      # Remove spekk-managed entries
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--loop-interval <minutes>` | `1440` (once a day) | How often to scan (≤ 60, or an exact multiple of 60 up to 1440) |
| `--consolidate-interval <minutes>` | `1440` (once a day) | How often to run consolidation (same rule) |

The default is one scan a day because a scan files at most three observations and then ends. A shorter interval does not make one run find more — it files up to three again each time it runs, so choose it based on how many observations you want to review, not on how thorough you want a run to be.

The installed entries run in the project directory, use `claude`'s absolute path (resolved at install time — fails clearly if `claude` isn't found), run headless (no TTY), guard against overlapping sessions via a project-scoped lock file, and append output to `.spekk/observer.log` / `.spekk/observer-consolidate.log`. `uninstall-cron` removes only the entries it added, identified by a `# spekk-observer` marker.

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

Manage cloud sandbox environments on DigitalOcean. The sandbox agent is a **generic Claude Code runner** — it is not spec-aware. It accepts prompts over a WebSocket connection from a control host and pipes them into `claude -p -`. For the full connection model, message protocol, and worker architecture, see the [Sandbox Architecture](./advanced/sandbox-architecture.md) doc.

!!! note "Post-creation registration"

    After `spekk sandbox create` provisions the VM, you must register the agent's token in the control host before the agent can connect. The `create` command prints the token and a reminder.

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

Skills resolve through layers: `.spekk/skills/<agent>/` (project), then `~/.config/spekk/skills/<agent>/` (user), then the skills built into the binary.

---

## `spekk install`

Install spekk agents (coach, builder, observer) into a coding assistant as subagents, plus the `spekk-dev-loop` orchestration skill.

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

### The `spekk-dev-loop` skill

Every install also writes the `spekk-dev-loop` skill — the outer coach → coordinate → builders → review pipeline — in whatever form the target uses for a reusable, agent-invokable workflow:

| Target | Written as | Location |
|--------|-----------|----------|
| `claude-code` | native skill | `~/.claude/skills/spekk-dev-loop/SKILL.md` |
| `opencode` | native skill | `~/.config/opencode/skills/spekk-dev-loop/SKILL.md` |
| `cursor` | `/spekk-dev-loop` command | `~/.cursor/commands/spekk-dev-loop.md` |
| `codex` | `/spekk-dev-loop` prompt | `~/.codex/prompts/spekk-dev-loop.md` (global only) |
| `copilot` | `/spekk-dev-loop` prompt | `.github/prompts/spekk-dev-loop.prompt.md` (`--project` only) |

**Automatic vs manual invocation:** Claude Code and OpenCode treat it as a native skill, so the model can invoke it on its own. Cursor, Codex, and Copilot expose it as a `/spekk-dev-loop` command the user triggers manually.

**Scope:** By default, the skill is installed globally. Pass `--project` to write it into the current repo instead. Two targets are exceptions: Copilot is always project-scoped (its personal prompts are IDE-managed), and Codex is always global.

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
