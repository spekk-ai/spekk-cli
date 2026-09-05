---
icon: lucide/square-terminal
---

# CLI Reference

Every spekk command, its flags, and what it does. `spekk help` prints the command list. `spekk <command> --help` prints the help for one command. `spekk next`, `spekk status`, and `spekk show` have no `--help` text of their own, so this page is their reference.

`spekk` with no command runs `spekk next`.

## `spekk init`

Set up a project for spec-driven development.

```bash
spekk init
```

Creates a `specs/` directory at the git root, or in the current directory when there is no repository. It writes `specs/README.md`. That file has a block that spekk manages: the concept model, the frontmatter fields, and a schema version. When `specs/` already exists, `spekk init` refreshes that block and keeps the text outside it as it is. When the README is missing, `spekk init` writes one. Run `spekk coach` next to draft your first spec.

## `spekk next`

Print the next assertion to work on, as JSON.

```bash
spekk next                               # Next on the current branch
spekk next --all-branches                # Next across all branches
spekk next --spec authentication         # Next in one spec
spekk next --assertion password-hashing  # One assertion, whatever its status
spekk next --all                         # The full spec hierarchy
spekk next --raw                         # Every parsed record, for scripts
spekk next --specs-dir ./my-specs        # Read a different specs directory
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--all` | | Print the full hierarchy: every spec, its assertions, and the observations |
| `--all-branches` | | Include assertions from all branches |
| `--spec <id>` | `-s` | Only assertions of this spec |
| `--assertion <id>` | | Print this assertion, whatever its status |
| `--raw` | | Print every parsed record, for downstream tools |
| `--specs-dir <path>` | | Read specs from this directory (default: `specs/` at the git root) |

**How it selects an assertion:**

1. It drops assertions with status `done` or `draft`, and every assertion whose parent spec is `draft`.
2. It keeps only assertions on the current git branch, unless you pass `--all-branches`. An assertion with no `branch` field is on `main`.
3. With `--spec`, it keeps only that spec's assertions.
4. It drops an assertion whose `depends-on` target is not `done`.
5. It drops an `in_progress` assertion that a builder holds with a fresh `locked-by` value. An old lock does not block.
6. It sorts by priority (1 first), then by `created` (oldest first), then by id, and prints the first one.

`--assertion <id>` skips these rules and prints that assertion.

Before it reads the specs, `spekk next` rebuilds `.spekk/index.db` when the index is stale, absent, or from an older schema. A spec or assertion file that it cannot parse is skipped. One warning line on stderr gives the count. Run `spekk validate` for the detail.

**Output:**

An assertion is a JSON object with `type: "assertion"` and these fields:

| Field | Description |
|-------|-------------|
| `id` | Assertion id |
| `parent` | Parent spec id |
| `file` | Path to the markdown file |
| `priority` | 1 (high), 2 (medium), 3 (low) |
| `status` | Current status |
| `branch` | Branch the assertion is on |
| `created` | The `created` timestamp |
| `dependsOn` | Prerequisite assertion id, when set |
| `lockedBy` | The builder lock, when set |
| `title` | The first heading of the file |
| `content` | The markdown body |
| `spec` | The parent spec: `id`, `file`, `title` |

When nothing is ready, the output has `type: "complete"`. When `specs/` has no specs, it has `status: "empty"`. A parse error that stops the whole tree prints `{"error": true, "message": ...}` and exits 1.

## `spekk list`

List assertions in a table, or as JSON, TSV, or CSV.

```bash
spekk list                          # Table: ID, STATUS, PRI, PARENT, TITLE
spekk list --long                   # Add the FILE column
spekk list --status draft           # Only one status
spekk list --json                   # Flat JSON array, for jq
spekk list --tsv                    # Tab-separated, lowercase header
spekk list --csv                    # RFC 4180 CSV with a header row
spekk list --specs-dir ./my-specs   # Read a different specs directory
spekk list --cross-branch --json    # Merge preview across branches
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--status <value>` | | Keep only this status: `not_started`, `in_progress`, `done`, `draft`, or `failed` |
| `--long` | `-l` | Add the FILE column to the table, TSV, and CSV output |
| `--json` | | JSON array, one object per assertion |
| `--tsv` | | Tab-separated values with a lowercase header |
| `--csv` | | RFC 4180 CSV with a header row |
| `--specs-dir <path>` | | Read specs from this directory (default: `specs/` at the git root) |
| `--cross-branch` | | List spec and assertion files changed on other branches. See [cross-branch mode](#cross-branch-mode) |
| `--branch-filter <glob>` | | In cross-branch mode, only branches that match the glob |
| `--assertions-only` | | Accepted for old scripts. It changes nothing: assertions are the default |

Rows are sorted by priority, then by id, in every format. The JSON output also carries `branch` and `depends_on`, which the table, TSV, and CSV do not show. `--json`, `--tsv`, and `--csv` exclude each other. `--status` and `--specs-dir` do not apply to `--cross-branch`.

## `spekk status`

Print an overview of every spec and assertion.

```bash
spekk status
```

Shows each spec with its assertions and a done count, then the totals by status, the completion percentage, and the next assertion `spekk next` would return. It has no flags.

## `spekk validate`

Check every spec and assertion under `specs/` against a fixed set of invariants.

```bash
spekk validate
spekk validate --specs-dir ./my-specs
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--specs-dir <path>` | Read specs from this directory (default: `specs/` at the git root) |

**Checks, all in one pass:**

- Frontmatter well-formedness. A malformed spec or assertion file is a failure here. `spekk next` skips such a file.
- Parent resolution: every assertion names a spec that exists.
- `depends-on` validity: kebab-case, the target exists, no self-reference, no cycles.
- No duplicate spec or assertion ids.
- Lock state: only `in_progress` may carry a `locked-by`, and it need not carry one.
- Parent specs carry no rolled-up `status` field. The field is absent, or it is the literal `draft`.
- A spec directory that has assertion files but no main spec file. The parser drops the whole directory, so each assertion in it is out of the queue.
- A path with the name `assertions` that is not a directory, or an `assertions/` directory that spekk cannot read. Each one drops the assertions of that spec with no message.

A spec directory with no `assertions/` directory is not a fault. It is a spec with no assertions, and it parses correctly.

**Warnings.** These go to stderr and do not change the exit code. A branch can be absent for a short time, and an old lock is not an incorrect spec tree:

- A `branch` value that no ref matches, on an assertion that is not `done` and not `draft`. `spekk next` selects the queue by this value, so it cannot find such an assertion. `validate` reports each different value one time, with a count.
- A `locked-by` value that is old. A lock shows that a builder session holds the assertion now. An old value, or a value with no date, names a session that stopped.

**Exit codes:**

| Result | Exit | Output |
|---|---|---|
| Valid | 0 | One-line summary on stdout |
| No `specs/` directory | 0 | `validate: 0 specs, 0 assertions OK` |
| Warnings only | 0 | Warnings on stderr; the check passes |
| Any violation | 1 | One failure line per violation (file and problem), sorted by file then message |

A malformed field fails the parse of the whole tree, not only its own file. One bad line on the default branch stops every command that rebuilds the index. Run `spekk validate` before you commit an edit to `specs/`. See [Validation in CI and pre-commit](ci.md).

### Skipped files

`spekk next`, `spekk list`, `spekk status`, and `spekk show` are permissive on purpose. Each one skips a spec file or an assertion file that it cannot parse, so one error does not stop the queue. Each one writes a single line to stderr to report this:

```
Warning: 3 spec files skipped and missing from the queue. Run "spekk validate" for detail.
```

The line is the same for any number of files. `spekk validate` gives the detail. It names each file and its fault.

The line goes to stderr, so `--json`, `--csv`, and `--tsv` output stays machine-readable.

A skipped file is not in the work queue. You lose that work until a person examines the tree. This is why the commands report the count.

## `spekk index`

Build `.spekk/index.db`, a SQLite index of the spec tree, for `spekk query`.

```bash
spekk index
spekk index --force
spekk index --specs-dir ./my-specs
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--force` | Drop every table and build the index again from scratch |
| `--specs-dir <path>` | Read specs from this directory (default: `specs/` at the git root) |

The markdown files stay the source of truth. The database is derived from them, and spekk adds it to `.gitignore`. You rarely need this command: `spekk next` and `spekk query` rebuild the index when it is stale, absent, or from an older schema. The schema version is stamped in the database, so an index from an older spekk version is rebuilt with no manual step.

## `spekk query`

Run a read-only `SELECT` against the index and print the result.

```bash
spekk query "SELECT status, COUNT(*) FROM assertions GROUP BY status"
spekk query "SELECT id, title FROM specs WHERE status = 'draft'" --json
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--json` | JSON array of objects |
| `--tsv` | Tab-separated values with a lowercase header |
| `--csv` | RFC 4180 CSV with a header row |

The default output is a padded table. The index is refreshed first, so the result reflects the current specs. Only `SELECT` and `WITH ... SELECT` statements are accepted, and the database is opened read-only, so a query cannot change it.

**Schema:**

| Table | Columns |
|---|---|
| `specs` | `id`, `title`, `status`, `priority`, `branch`, `file` |
| `assertions` | `id`, `parent_id` (references `specs.id`), `title`, `status`, `priority`, `branch`, `file` |
| `depends_on` | `assertion_id`, `depends_on_id` (both reference `assertions.id`) |
| `frontmatter_fields` | `owner_type` (`spec`, `assertion`, or `observation`), `owner_id`, `key`, `value` |
| `observations` | `slug`, `ref`, `type`, `severity`, `status`, `created`, `announced`, `pr`, `title`, `file`. One row per (slug, ref) |
| `observation_files` | `slug`, `ref`, `path`. One row per `affected` path |

`depends_on` holds assertion-level edges only. A spec-level relationship is prose in the spec body, not data.

### Custom frontmatter fields

A top-level frontmatter key outside the known set (`id`, `parent`, `created`, `priority`, `status`, `branch`, `depends-on`, `locked-by`) is a custom field. `spekk validate` accepts it without a warning, and the index stores it in `frontmatter_fields`, one row per distinct value. Three multi-value spellings index the same way: a flow sequence (`tags: [infrastructure, hipaa]`), a comma-separated scalar (`workflows: w1-note-and-claim, w2-claim-reimbursement`), and a YAML block list (`- item` lines).

Value rules:

- In a scalar or a flow sequence, an unquoted comma splits the value. A quoted region protects its commas: `note: "Hello, world"` is one value, and `[a, "b, c"]` is two.
- A block-list item is never split again. It keeps its commas as written.
- Comment lines, nested-map children, empty keys, and block scalars (`key: |` or `key: >`) never become custom fields. The frontmatter parser reads top-level scalars, flow sequences, and flat block lists only.
- A trailing `# ...` that follows whitespace is a comment. It is cut before the value is read, on a key line and on a block-list item. A comment on a key that opens a list (`tags: # my tags`) keeps the list. An item that holds only a comment (`- # placeholder`) is an empty item, not the end of the list.
- A `#` with no space before it is data, so `link: https://example.com/x#frag` keeps its fragment. A quote protects a `#` only when the quote opens the value: `note: "a # b"` keeps its hash, and `apos: it's fine # note` loses its comment, because the apostrophe is plain text and not the start of a quoted scalar.
- The top level is the shallowest column in the block, not column zero, so a root mapping that is indented as one block still parses. A deeper line is a nested child only when a key above it opened a region: a key with an empty value, or a block scalar. Such a child sets nothing. A `priority:` inside a prose block is not the assertion's priority. A deeper line with no such key above it is a stray indent, and it reads as a top-level key.

An observation carries custom fields under the same rule, with its own known set (`slug`, `type`, `severity`, `status`, `created`, `announced`, `pr`, `affected`). Its rows use `owner_type = 'observation'` and the slug as `owner_id`. `affected` is a known key and never becomes a custom field. It is the evidence gate, and `observation_files` is its table.

The `owner_id` of an observation is the slug alone, although `observations` and `observation_files` are keyed by (slug, ref). Each observer branch is cut from main and inherits every observation already merged, so the same slug reaches the index once per ref. The custom-field rows are merged across refs. A slug carried by twenty branches indexes the same as a slug carried by one.

So query `frontmatter_fields` alone. A join to `observations` on the slug returns one copy of every field row per ref. Add `DISTINCT` when you cannot avoid the join.

```sql
SELECT owner_id AS slug, key, value
  FROM frontmatter_fields
 WHERE owner_type = 'observation' AND key = 'skill';
```

!!! warning "`depends-on` is a chain, not a list"

    A list here is refused, and the error fails the parse of the whole tree. One bad line stops every command that rebuilds the index, for every branch and every user, until it is fixed.

    ```yaml
    depends-on: [first-step, second-step]   # refused: breaks every index rebuild
    depends-on: first-step                  # correct: one id, chain the rest
    tags: [infrastructure, hipaa]           # correct: a custom field may be a list
    ```

    The single id is deliberate. It controls the order in which work reaches builders: a chain releases assertions one at a time, and assertions with no link between them can run in parallel. A set of prerequisites carries no such order, so the coach's coordinator skill turns the set into a chain.

Per-tag reporting is one query:

```bash
spekk query "SELECT f.value AS workflow, COUNT(*) AS total, SUM(a.status = 'done') AS done
  FROM assertions a
  JOIN frontmatter_fields f
    ON f.owner_type = 'assertion' AND f.owner_id = a.id AND f.key = 'workflows'
  GROUP BY f.value"
```

## `spekk show`

Open the spec explorer, a web page that shows the spec tree.

```bash
spekk show                                          # Write the page and open it
spekk show --watch                                  # Serve it and reload on change
spekk show --cross-branch                           # Merge preview
spekk show --cross-branch --branch-filter 'feat/*'  # Only branches that match the glob
spekk show -w --cross-branch                        # Merge preview with live reload
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--watch` | `-w` | Serve the page on `127.0.0.1` and reload it when a spec file changes |
| `--cross-branch` | | Read-only preview of what a merge of each other branch would do to the specs |
| `--branch-filter <glob>` | | In cross-branch mode, only branches that match the glob (for example `'feat/*'`) |

Without `--watch`, `spekk show` writes `.spekk/index.html` and opens it in your browser. With `--watch`, it serves the page from a local port and reloads it when a spec file changes. Set the `CI` environment variable to stop watch mode from opening a browser.

The page shows:

- The spec and assertion tree, with status and priority badges
- A metro map of the dependency chains
- Drag to pan, click to open a spec or an assertion
- Done specs hidden by default, with a toggle to show them
- A search box that filters by name, status, or priority

### Cross-branch mode

`--cross-branch` answers one question: what would a merge of each other branch into my current branch do to the specs? It scans every local and remote-tracking branch, classifies each changed spec and assertion file into one of four states, and shows the result in the explorer. It does not touch the working tree or the git index.

#### The four states

| Badge | State | Meaning |
|-------|-------|---------|
| `+` | Incoming addition | The file exists on another branch and not on yours. Shown as a "foreign" item with a striped background |
| `↻` | Incoming modification | The file changed on another branch and not on yours, so it would merge cleanly. Shows a status change, for example `not_started` to `done` |
| `⚠` | Conflict | The file changed on both branches in ways that do not merge. Red border when confirmed (git 2.38 or later), orange dashed border when not confirmed |
| `✕` | Incoming deletion | The file is deleted on another branch and still exists on yours. Shown with a struck-through title |

When a spec has contributions in more than one state, the summary badge shows the state with the highest precedence: conflict, then deletion, then addition, then modification.

#### Branch discovery and filtering

Cross-branch mode compares against local branches (`refs/heads/*`) and remote-tracking branches (`refs/remotes/*`). It excludes the current branch and symbolic refs such as `origin/HEAD`. A branch with the same name in both namespaces counts one time, and the local one wins.

Use `--branch-filter` to narrow the scope:

```bash
spekk show --cross-branch --branch-filter 'feat/*'    # Only feature branches
spekk show --cross-branch --branch-filter 'hotfix/*'  # Only hotfix branches
```

The filter matches the branch name with the remote prefix removed. `origin/feat/login` matches as `feat/login`, so `feat/*` covers both local and remote-tracking branches, and `origin/*` matches nothing.

The filter uses Go `filepath.Match` glob rules: `*` matches characters other than a separator, `?` matches one character, and `[...]` matches a character class. A malformed pattern is an error.

#### Page elements

- A branch banner at the top that lists the compared branches
- A branch checkbox dropdown. Deselect a branch to hide its contributions; the badges recalculate. The selection is kept per project in the browser's `localStorage`
- Inline badges on the affected specs and assertions in the tree
- Contribution details in the detail panel: the state per branch, and the status change
- Foreign items (incoming additions) with a striped background and a dimmed title. They disappear when every contributing branch is deselected

#### Git version

Cross-branch mode uses `git merge-tree --write-tree` (git 2.38 or later) for three-way conflict detection. It probes the git version one time per process.

On an older git, the mode still works, with less certainty:

- Additions, modifications, and deletions are classified as usual
- A conflict cannot be confirmed. A file changed on both sides is marked as a possible conflict, with `⚠` in an orange dashed border
- An orange banner says that conflict detection is not available

#### Read-only guarantee

Every git operation goes through one allowlist that permits a small set of read-only subcommands (`rev-parse`, `for-each-ref`, `merge-tree`, and a few others). No checkout, merge, index change, or ref change can occur.

#### Machine-readable output with `spekk list --cross-branch`

The same classification is available as data for agents and scripts. The explorer stays the page for people.

```bash
spekk list --cross-branch --json
spekk list --cross-branch --branch-filter 'feat/*' --json
spekk list --cross-branch --tsv
```

One row per changed (file, branch) pair, with columns `path`, `branch`, `state` (`incoming_add`, `incoming_mod`, `conflict`, or `incoming_del`), `degraded` (a conflict that git older than 2.38 could not confirm), and `old_status` and `new_status` (the status change on a clean incoming modification, for example `not_started` to `done`; empty otherwise). The default output is a table. `--json` prints an array of objects, where every row carries every key and an empty set prints `[]`. `--tsv` and `--csv` print the same columns. An observer can use this to flag an assertion that is `done` on `main` and that a feature branch moves back to `in_progress`, before the branch merges. The same read-only guarantee applies.

#### Watch mode with cross-branch

With `--watch`, the cross-branch classification is cached on a fingerprint of the git ref state:

- An edit to a spec file in the working tree triggers a cheap re-render with the cached classification
- A ref change (a new commit, a fetch, a branch move) invalidates the cache and triggers a full reclassification
- Two watchers run: one polls the spec files, and one polls the git refs
- A transient git error is logged one time, and the watcher recovers when git is available again

## `spekk builder`

Launch the builder agent in Claude Code to implement assertions.

```bash
spekk builder                        # Build assertions until you stop it
spekk builder --once                 # Build one assertion, then exit
spekk builder --dry-run              # Print the next assertion, build nothing
spekk builder --spec authentication  # Only this spec's assertions
spekk builder --assertion login      # Build one assertion, whatever its status
spekk builder --confirm              # Ask before each build
spekk builder --interactive          # Work with the builder in one session
spekk builder my-skill               # Launch with a builder skill active
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| *(none)* | | Build assertions in a loop until you press Ctrl+C |
| `--once` | | Build one assertion, then exit |
| `--dry-run` | `-d` | Print the next assertion and exit. Claude Code does not start |
| `--spec <id>` | `-s` | Only assertions of this spec |
| `--assertion <id>` | | Build this assertion, whatever its status |
| `--confirm` | `-c` | Ask `y/n/q` before each build: build, skip, or quit |
| `--interactive` | `-i` | Start Claude Code with the builder prompt as the system prompt, and wait for your input |

The builder needs the `claude` command on your `PATH`. Each build runs `claude --dangerously-skip-permissions` with the builder prompt and the assertion. Each loop iteration:

1. Runs `spekk next`, with your `--spec` or `--assertion` filter, to get the assertion.
2. Launches Claude Code with the builder prompt. The agent reads the assertion, writes tests where they apply, implements the change, runs the tests, runs `spekk validate`, and commits.
3. Waits one second and repeats, until `spekk next` reports nothing ready. It then waits five seconds and asks again.

In `--once` mode the exit code is 0 when the build succeeded and 1 when it did not. A skill name as the first positional argument launches one Claude Code session with that skill active, and no loop.

??? info "Lean testing"

    The builder prompt asks for lean tests:

    - Test behavior, not implementation detail
    - One test per behavior
    - Delete tests that add nothing
    - No tests for trivial code
    - Prefer an integration test to a unit test where it fits

## `spekk coach`

Launch the coach agent in Claude Code to write specs.

```bash
spekk coach                          # Interactive spec creation
spekk coach meeting                  # Turn a meeting transcript into specs
spekk coach meeting notes.txt        # The same, from a file
spekk coach coordinate               # Plan dependencies and branches
spekk coach validate                 # Assess a business idea
```

The coach needs the `claude` command on your `PATH`. The first positional argument names a skill. `spekk coach --help` lists the skills it can find. See [Skills](coach-skills.md) for what each built-in skill does and how to add your own.

### `spekk coach`

Helps you write a well-formed spec:

- Asks questions about the requirement
- Writes the spec and assertion files
- Keeps the format and the structure correct
- Commits the change

### `spekk coach meeting [file]`

Turns a meeting transcript into three outputs:

- Action items, appended to `TODOS.md`
- Specs, as files in `specs/`
- Decisions, appended to `CONTEXT.md`

With no file, the coach asks you to paste the transcript.

### `spekk coach coordinate`

Makes a dependency-aware work plan:

1. Reads every `draft` and `not_started` assertion
2. Finds the prerequisites
3. Groups related work into feature branches
4. Shows the dependency tree for your approval
5. Writes `depends-on` and `branch` into the frontmatter
6. Runs `spekk validate`
7. Commits the change

## `spekk observer`

Find drift between the specs and the code, one observation at a time.

```bash
spekk observer                    # Run one scan
spekk observer --quiet            # Ask the agent for less output
spekk observer coverage-gap       # Run one skill
spekk observer prune              # Find unused code and redundancy, as recommendations
spekk observer consolidate        # Curate the open observations
spekk observer digest             # Print the open observations, ranked
spekk observer scan-check ...     # Check a finding against suppressions and dedup
spekk observer announce           # Announce open observations (sandbox only)
spekk observer install-cron       # Schedule the observer with crontab
spekk observer uninstall-cron     # Remove the crontab entries
```

**Flags for a scan or a skill run:**

| Flag | Description |
|------|-------------|
| `--quiet` | Ask the agent for less output. The agent decides what that means |
| `--headless` | Run Claude Code with `-p`, with no terminal. `install-cron` sets this |
| `--claude-path <path>` | The `claude` binary to run. `install-cron` sets this, because cron has a short `PATH` |

`--interval` is no longer a flag. Passing it is an error that names the replacement, `install-cron --loop-interval`.

The observer detects a change to the code that the specs do not record, and a change to the specs that the code does not implement. A run files one observation. It searches until it finds drift, files it on an `observer/<slug>` branch, and stops. The drift it did not reach is still there tomorrow, so the second finding is the next run's first. Run it again to continue. A run says which areas it did not reach, so a short run does not read as a clean result.

A run reports a short summary from the rendered digest (`spekk observer digest`). There is no committed digest file. When the digest is empty, the observer says nothing.

How many observations you receive depends on how often you run it, not on how long a run goes on. Set that with `install-cron`, or with the scheduler you already use.

**Skills.** The observer resolves skills the same way the coach and the builder do: `.spekk/skills/observer/`, then `~/.config/spekk/skills/observer/`, then the skills in the binary. `spekk observer --help` lists them. The built-in skills:

| Skill | Description |
|-------|-------------|
| `coverage-gap` | Finds code that a spec could document. It is an aid for gradual adoption, not a defect report. Code without a spec is normal |
| `prune` | Finds code that nothing uses, and design-level redundancy: duplication, over-abstraction, dead configuration. It recommends and never deletes. A missing spec is not a signal |
| `consolidate` | Curates the open observations by editing frontmatter on their own branches, for example `open` to `dismissed`. It writes no digest file |

### `spekk observer digest`

Print the open observations that are live claims, ranked by severity (high, then medium, then low; oldest first within a severity), capped at five.

```bash
spekk observer digest
spekk observer digest --json
```

A live claim is the observation read from the branch named after it, whose slug has not reached `main`. A copy that another branch inherited is not a claim. A slug already on `main` is resolved. The digest reads committed git state only, with no checkout and no remote call. Run `git fetch` first when you want the remote observer branches current.

### `spekk observer scan-check`

Check a finding against the suppression file and the open observations before the observer files it. The observer prompt runs this; you rarely run it by hand.

```bash
spekk observer scan-check --type code_spec_misalignment --slug parser-skips-empty-file --affected internal/parser/parser.go
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--type <type>` | `code_spec_misalignment` or `outdated_specs` (required) |
| `--slug <slug>` | The kebab-case slug the observation would get (required) |
| `--affected <paths>` | Comma-separated evidence paths (required) |

The result is one JSON line:

| `result` | Meaning |
|---|---|
| `suppressed` | An active entry in `.spekk/dont-flag.yaml`, as committed on `main`, matches an evidence path or the slug. File nothing |
| `covered` | A live claim with the same type and slug already exists. File nothing |
| `clear` | File the observation with the returned `slug`. When an observation on `main` already has the plain slug, the returned slug carries a `-YYYYMMDD` suffix |

A malformed `.spekk/dont-flag.yaml` fails the check with a message that names the entry. A broken suppression file is never read as empty. See [Suppressing observations](configuration.md#suppressing-observations) for the file format.

### `spekk observer announce`

Announce the open observations on the connected chat surface. This command runs inside a sandbox session only, because it delivers through the session's conversation spool.

```bash
spekk observer announce
```

One run: `git fetch` (the only remote read), refresh the index, pick the unannounced open observations with severity high or medium (low never announces), high first and oldest first, from `observer/*` branches on `origin`. It opens one conversation with at most three findings, then commits an `announced:` timestamp to each observer branch and pushes. An observation with no `affected` path never announces. With nothing to announce it prints `nothing to announce` and exits 0.

When `SPEKK_CONVERSATION_SPOOL` is not set, the command fails, appends a line to `.spekk/observer-conversation.log`, and exits non-zero. Every other failure does the same, and leaves `announced:` unset, so the next run retries.

### `spekk observer install-cron`

Install two crontab entries that run the observer on a schedule.

```bash
spekk observer install-cron                              # Once a day
spekk observer install-cron --loop-interval 60           # Scan every hour
spekk observer install-cron --consolidate-interval 720   # Consolidate every 12 hours
spekk observer uninstall-cron                            # Remove the entries
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--loop-interval <minutes>` | `1440` (once a day) | How often to scan. Accepts 60 or less, or an exact multiple of 60 up to 1440 |
| `--consolidate-interval <minutes>` | `1440` (once a day) | How often to run `consolidate`. Same rule |

Values are space-separated. `--loop-interval=60` is refused. A run files one observation whatever the interval, so this flag sets how many observations arrive, not how thorough a run is. Once a day is a rate a person can keep up with. Shorten it to work through drift faster.

Each entry changes into the project directory, runs the observer with `--headless` and the absolute path of `claude` (found at install time; the command fails with a message when `claude` is not on the `PATH`), and appends output to `.spekk/observer.log` or `.spekk/observer-consolidate.log`. A lock file per skill under `.spekk/` stops two runs of the same skill from overlapping. The consolidation entry runs 30 minutes after the scan, so it curates what the scan filed. Each line ends with a `# spekk-observer` marker, and `uninstall-cron` removes only lines with that marker.

## `spekk loop`

Run an agent in a loop that commits after each session.

```bash
spekk loop builder    # Build, commit, repeat
spekk loop coach      # Coach session, commit specs, repeat
```

`spekk loop builder` runs `spekk next`, launches the builder agent on the result, stages every change with `git add .`, commits it as `Complete <id>`, and repeats. When nothing is ready it waits five seconds and asks again. `spekk loop coach` launches the coach agent, and when the session ends it commits any change under `specs/` and starts a new session. Press Ctrl+C to stop either loop.

`spekk builder` on its own also loops. The difference is the commit: `spekk builder` leaves the commit to the agent, and `spekk loop builder` commits everything in the working tree after each session.

## `spekk serve`

Start the WebSocket server that the spekk browser extension connects to.

```bash
spekk serve                        # localhost:3118
spekk serve --port 8080            # Another port
spekk serve --host 0.0.0.0         # Every interface
spekk serve --verbose              # Log each WebSocket message
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--port <port>` | `-p` | Port to listen on (default: `3118`) |
| `--host <host>` | | Host to bind to (default: `localhost`) |
| `--verbose` | `-v` | Log each WebSocket message |

The server accepts connections from a `chrome-extension://` origin and from `localhost`. It launches agents on request from the extension.

## `spekk sandbox`

Manage sandboxes: machines that run the spekk agent client. spekk can create the machine as a DigitalOcean droplet, or register a machine you already have. The agent client on the machine is a generic Claude Code runner. It knows nothing about specs. It connects out to a control host over WebSocket, receives prompts, and pipes them into `claude -p -`. See [Sandbox Agent Architecture](advanced/sandbox-architecture.md) for the connection model and the message protocol.

Every sandbox command reads its settings from environment variables. See [Sandbox provisioning](configuration.md#sandbox-provisioning) for the list, and for the `--auth` modes.

!!! note "Register the token"

    After `create` or `provision` finishes, one manual step remains. Register the agent token that the command prints on the control host. Until you do, the agent tries to connect and fails authentication.

### `spekk sandbox create`

Create a sandbox.

```bash
spekk sandbox create --name my-sandbox
spekk sandbox create --name my-sandbox --region sfo3 --size s-4vcpu-8gb
spekk sandbox create --name my-sandbox --project "My Project" --vpc <uuid>
spekk sandbox create --name my-sandbox --provision-timeout 45m
spekk sandbox create --name my-sandbox --auth subscription
spekk sandbox create --name my-box --ip 203.0.113.10 --ssh-key ~/.ssh/my-box
spekk sandbox create --name my-box --ip 203.0.113.10 --ssh-key ~/.ssh/my-box --ssh-user ubuntu
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--name <name>` | Sandbox name (required). Lowercase letters, digits, and hyphens, starting with a letter or a digit |
| `--provider <provider>` | `digitalocean` (default), or `none` for a machine you already have. `--ip`, `--ssh-key`, or `--ssh-user` implies `none` |
| `--auth <mode>` | How the agent authenticates Claude: `bedrock` (default) or `subscription` |
| `--region <region>` | DigitalOcean region (default: `nyc1`) |
| `--size <size>` | Droplet size slug (default: `s-2vcpu-4gb`) |
| `--project <project>` | Assign the droplet to a DigitalOcean project, by name or UUID |
| `--vpc <uuid>` | Place the droplet in a DigitalOcean VPC |
| `--provision-timeout <duration>` | How long to wait for cloud-init, as a Go duration (default: `30m`) |
| `--ip <address>` | Address of a machine you already have |
| `--ssh-key <path>` | Private key that reaches that machine as the login user |
| `--ssh-user <user>` | SSH login user for that machine (default: `root`). A non-root user must have passwordless sudo |

The DigitalOcean flags and the existing-machine flags exclude each other. `create` refuses a name that is already recorded: destroy the old sandbox first.

**A DigitalOcean sandbox.** `create` checks that the environment variables for the auth mode are set, downloads the agent binary and the cloud-init template from the latest spekk release, generates an SSH key pair under spekk's config directory, creates the droplet with cloud-init, and records it. It then waits for SSH and for cloud-init to write `/opt/spekk/.provisioned`. While it waits, it prints a progress line at most once a minute with the time waited and the last line of `/var/log/cloud-init-output.log`. When `cloud-init status` reports `error`, or `done` without the marker, the wait stops at once. When the marker appears, `create` injects the credentials, configures git for the agent, deploys the agent binary, marks the sandbox `active`, and prints the agent token.

When the wait runs out, the droplet keeps running and the record stays at `provisioning`. Nothing is destroyed. Finish it with [`spekk sandbox provision`](#spekk-sandbox-provision-name) when cloud-init is done, or remove it with `destroy`.

**A machine you already have.** spekk does not provision a machine it did not create. Prepare the machine yourself, so that it carries `/opt/spekk/.provisioned`, an `agent` user, and the directories the agent needs. `create` checks the marker over SSH, then runs the same three steps: inject the credentials, configure git, deploy the agent. With `--ssh-user`, the three steps and the later teardown run under `sudo`. Everything else runs as the login user.

### `spekk sandbox provision <name>`

Finish a sandbox that `create` left at `provisioning`.

```bash
spekk sandbox provision my-sandbox
spekk sandbox provision my-sandbox --force              # A sandbox that is not at "provisioning"
spekk sandbox provision my-sandbox --auth subscription  # Change the auth mode
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--auth <mode>` | | `bedrock` or `subscription`. Default: the mode the sandbox was created with |
| `--force` | `-f` | Provision a sandbox whose status is not `provisioning` |

`provision` checks that the environment variables for the auth mode are set, checks that `/opt/spekk/.provisioned` exists on the machine, and then runs the same three steps `create` runs after its wait, in the same order and through the same code. It marks the sandbox `active` and prints a new agent token to register. A sandbox recorded before 1.28.0 has no auth mode on record, and reads as `bedrock`.

### `spekk sandbox list`

Print every recorded sandbox: name, IP, region, status, and creation time.

```bash
spekk sandbox list
```

### `spekk sandbox status <name>`

Print the stored fields of a sandbox, the live machine state, and two SSH checks: the provisioned marker, and the `spekk-agent` service.

```bash
spekk sandbox status my-sandbox
```

When the cloud API cannot be reached, or the provider cannot be built because its token is missing, `status` prints a warning and shows the stored status marked `(stored)`. For a machine spekk did not create there is no live state, and the stored status is shown with no warning.

### `spekk sandbox ssh <name> [ssh-flags...]`

Open an SSH session to a sandbox as its login user. Extra arguments go to `ssh`.

```bash
spekk sandbox ssh my-sandbox
spekk sandbox ssh my-sandbox -L 8080:localhost:8080
spekk sandbox ssh my-sandbox tail -f /var/log/cloud-init-output.log
```

### `spekk sandbox destroy <name>`

Tear down a sandbox and remove its local record.

```bash
spekk sandbox destroy my-sandbox
spekk sandbox destroy my-sandbox --force   # No confirmation prompt
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--force` | `-f` | Skip the confirmation prompt, and remove the record when the teardown cannot run |

For a DigitalOcean sandbox, `destroy` deletes the droplet and its SSH key on DigitalOcean, then removes the local record. When the record names no droplet, `destroy` refuses, because the droplet may still be running and billing. `--force` removes the record anyway.

For a machine spekk did not create, `destroy` stops the `spekk-agent` service, removes the credential files spekk put on the machine, and removes the local record. It makes no cloud API call and needs no API token. When the teardown fails, `destroy` keeps the record, because removing it would leave an agent running with live credentials and nothing that points at it. `--force` removes the record anyway.

The local SSH key pair is deleted only when spekk generated it. The test is where the private key is: the recorded path, cleaned and made absolute, must be inside the `keys` directory under spekk's config directory, and so must the file it resolves to after symlinks are followed on both sides. A key anywhere else is kept, and `destroy` prints one line to stderr that names the path it kept.

### `spekk sandbox deploy <name>`

Download the agent binary from the latest spekk release and install it on a sandbox.

```bash
spekk sandbox deploy my-sandbox
```

It copies the binary, writes the `spekk-agent` systemd unit, and restarts the service. On a root login the copy goes straight to `/opt/spekk/agent-client`, and `scp` cannot overwrite a binary that is running. See [Cutting a release](releasing.md#known-sharp-edges).

## `spekk conversation open`

Ask for a conversation with a person on the connected chat surface. This command works inside a sandbox session only.

```bash
spekk conversation open --title "Need a decision on the auth flow" --body "The spec says OAuth. The code uses JWT." --severity warning
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--title <text>` | Short summary (required) |
| `--body <text>` | Full details (required) |
| `--severity <level>` | `info` (default), `warning`, or `critical` |

The command writes one request file into the spool directory named by `SPEKK_CONVERSATION_SPOOL`, and exits. It does not wait for a reply. The agent client on the sandbox sets that variable for each Claude session, reads the file, and sends a `conversation_open` frame to the control host. Outside a sandbox session the variable is not set, and the command fails with a message that says so.

## `spekk prompt <agent>`

Print an agent's resolved prompt to stdout.

```bash
spekk prompt coach
spekk prompt builder
spekk prompt observer
```

The prompt is resolved through the layers in [Configuration](configuration.md#prompt-customization): a `.spekk/` or `~/.config/spekk/` override, then the prompt in the binary, then the extensions. The output is the prompt `spekk <agent>` launches with. The agent files that `spekk install` writes run this command at session start.

## `spekk skill`

List and print agent skills.

```bash
spekk skill list coach                     # Each skill and the directory it came from
spekk skill show coach coordinator-skill   # Print a skill
```

Skills resolve through layers: `.spekk/skills/<agent>/` (project), then `~/.config/spekk/skills/<agent>/` (user), then the skills in the binary.

## `spekk skills list <agent>`

List every skill an agent can use, with its source.

```bash
spekk skills list coach
```

This is the same list as `spekk skill list`, with a header line and `(embedded)` for a skill in the binary.

## `spekk install --target <tool>`

Install the spekk agents into a coding assistant.

```bash
spekk install --target claude-code   # alias: claude
spekk install --target copilot
spekk install --target cursor
spekk install --target opencode
spekk install --target codex
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--target <tool>` | The host tool (required): `claude-code`, `copilot`, `cursor`, `opencode`, or `codex` |
| `--project` | Install into the repository that holds the working directory, instead of your home directory |

The install writes the observer as an agent, and the coach, the builder, and the dev loop as skills. The observer, coach, and builder files are thin: each one runs `spekk prompt <role>` at session start, so it does not go stale. The `spekk-dev-loop` skill has its content in the file, so a new spekk version needs a new `spekk install`. `spekk update` warns you when an installed file no longer matches the binary. `spekk install` also removes the coach and builder agent files that a version before 1.15.0 wrote.

**Where the files go:**

| Target | Agents (global) | Agents (`--project`) | Skills (global) | Skills (`--project`) |
|--------|-----------------|----------------------|-----------------|----------------------|
| `claude-code` | `~/.claude/agents/` | `.claude/agents/` | `~/.claude/skills/<name>/SKILL.md` | `.claude/skills/<name>/SKILL.md` |
| `opencode` | `~/.config/opencode/agents/` | `.opencode/agents/` | `~/.config/opencode/skills/<name>/SKILL.md` | `.opencode/skills/<name>/SKILL.md` |
| `cursor` | `~/.cursor/agents/` | `.cursor/agents/` | `~/.cursor/commands/<name>.md` | `.cursor/commands/<name>.md` |
| `codex` | `~/.codex/prompts/` | not supported | `~/.codex/prompts/<name>.md` | not supported |
| `copilot` | `~/.copilot/agents/` | `.github/agents/` | none; the coach and builder are agents | `.github/prompts/<name>.prompt.md` |

Claude Code and OpenCode read the skills as native skills, so the model can invoke them. Cursor, Codex, and Copilot get them as `/spekk-coach`, `/spekk-builder`, and `/spekk-dev-loop` commands that you invoke. A `--project` install writes into the repository root, from any directory inside it, the same root `spekk init` uses. Outside a repository it writes into the working directory.

**A managed path belongs to spekk.** Each install brings every file in the table to the current content. A file with its stamp intact, which is spekk's own content from another version, is replaced with no backup and no message. Every other file is copied to `<path>.bak` first, and the install reports the path on stderr. When `<path>.bak` already holds a different version, the copy goes to `<path>.bak.1`, then `<path>.bak.2`, and so on. A backup that already holds the same bytes is kept as it is. A file that an older spekk wrote before stamps existed is backed up too, because spekk cannot prove it is unchanged. To keep a local variant of a skill, give it your own name instead of editing the managed file.

**A managed path spekk cannot read is reported, not forced.** When the path holds something that is not a regular file, or a file spekk cannot open, `spekk install` leaves it alone and says so, and `spekk update` reports it. Check the file and its permissions. One such path does not stop the run: the other files are installed and checked.

**A symlink at a managed path is a conflict spekk does not settle.** When the path is a symlink, for example from a dotfiles manager, two tools own one file. spekk writes nothing, removes nothing, and reports the path and its target. Only you can say which tool owns the path. The test is on the file itself: when a parent directory is a symlink, the files inside it are ordinary files, and spekk writes them.

**Other tools.** An assistant that can run a shell command needs no installer. Point it at `spekk prompt coach` or `spekk prompt builder` through its custom-agent or rules mechanism. When it reads `AGENTS.md`, one line there is enough: run `spekk prompt <agent>` for spec-driven work, and follow the output.

## `spekk install <agent> <skill>`

Install a skill from the skill registry, [`github.com/spekk-ai/spekk-skills`](https://github.com/spekk-ai/spekk-skills), or from a URL.

```bash
spekk install coach meeting-notes                                  # Into .spekk/skills/coach/
spekk install coach meeting-notes --global                         # Into ~/.config/spekk/skills/coach/
spekk install coach my-skill --source https://example.com/skill.md # From a URL
spekk install coach meeting-notes --force                          # Overwrite a file that exists
spekk install --list coach                                         # What the registry has for the coach
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--local` | Write to `.spekk/skills/<agent>/` in the working directory (default) |
| `--global` | Write to `~/.config/spekk/skills/<agent>/` |
| `--source <URL>` | Fetch the skill from this URL instead of the registry |
| `--force` | Overwrite a file at the destination |
| `--list <agent>` | List the registry's skills for an agent, and install nothing |

`<agent>` is `coach`, `builder`, or `observer`. A skill name is one path segment with no separator. The install refuses to overwrite a file that exists unless you pass `--force`, and it checks that before it fetches anything. Two environment variables point the command at a mirror; see [Skill registry](configuration.md#skill-registry).

## `spekk uninstall <agent> <skill>`

Remove a skill that `spekk install <agent> <skill>` wrote.

```bash
spekk uninstall coach meeting-notes            # From .spekk/skills/coach/
spekk uninstall builder my-skill --global      # From ~/.config/spekk/skills/builder/
```

Pass `--global` or `--local` to match where the skill is. `--local` is the default.

## `spekk update`

Replace the running binary with the latest GitHub release.

```bash
spekk update           # Install the latest release
spekk update --check   # Print the current and latest version, install nothing
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--check` | `-c` | Report whether an update exists, and install nothing |

The update writes to the install directory, so that directory must be writable. See [Install](install.md#updating). A development build (`spekk version` prints `dev`) cannot update. After the check or the install, `spekk update` scans the files `spekk install --target` wrote and reports each one that no longer matches the binary, with the command to run.

## `spekk version`

Print the version. `spekk --version` does the same.

## `spekk help`

Print the command list. `spekk --help` and `spekk -h` do the same.

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Error. The message on stderr, or the JSON on stdout, says what went wrong |

`spekk validate` exits 1 on any violation. `spekk builder --once` exits 1 when the build did not succeed.
