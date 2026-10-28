---
icon: lucide/square-terminal
---

# CLI Reference

Every spekk command, its flags, and what it does. `spekk help` prints the command list. `spekk <command> --help` prints the help for one command. `spekk next` and `spekk status` have no `--help` text of their own, so this page is their reference.

`spekk` with no command runs `spekk next`.

For app-managed sandbox commands and key setup, see [App-managed sandboxes](app-managed-sandboxes.md).

Two rules hold for every command. A flag that takes a value takes it after a space, so `--flag=value` is an unknown argument and fails. A flag that takes a value is supplied once: a repeat fails rather than discard the value you typed. A flag that takes no value may repeat, because a repeat discards nothing.

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

When no assertion is open (all are `done`), the output has `type: "complete"`. When open assertions remain but none is ready, the output has `type: "none_eligible"`, with `remaining` (the number of open assertions), `reasons` (a count for each of `other_branch`, `blocked`, `locked` and `draft`), and a `message` that names them. An assertion with no `branch` field belongs to `main`, so on another branch it counts as `other_branch`: run `spekk next --all-branches` to see it. When `specs/` has no specs, it has `status: "empty"`. A parse error that stops the whole tree prints `{"error": true, "message": ...}` and exits 1.

## `spekk list`

List assertions in a table, or as JSON, TSV, or CSV.

```bash
spekk list                          # Table: ID, STATUS, PRI, PARENT, TITLE
spekk list --long                   # Add the FILE column
spekk list --status draft           # Only one status
spekk list --status done --priority 1 # Combine status and priority filters
spekk list --json                   # JSON object with an assertions array
spekk list --tsv                    # Tab-separated, lowercase header
spekk list --csv                    # RFC 4180 CSV with a header row
spekk list --specs-dir ./my-specs   # Read a different specs directory
spekk list --cross-branch --json    # Merge preview across branches
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--status <value>` | | Keep only this status: `not_started`, `in_progress`, `done`, `draft`, or `failed` |
| `--priority <N>` | | Keep only this priority. Requires a nonnegative integer and works with `--status` |
| `--long` | `-l` | Add the FILE column to the table, TSV, and CSV output |
| `--json` | | JSON object with an `assertions` array |
| `--tsv` | | Tab-separated values with a lowercase header |
| `--csv` | | RFC 4180 CSV with a header row |
| `--specs-dir <path>` | | Read specs from this directory (default: `specs/` at the git root) |
| `--cross-branch` | | List spec and assertion files changed on other branches. See [cross-branch mode](#cross-branch-mode) |
| `--branch-filter <glob>` | | In cross-branch mode, only branches that match the glob |
| `--assertions-only` | | Accepted for old scripts. It changes nothing: assertions are the default |

Rows are sorted by priority, then by id, in every format. The JSON output also contains `branch` and `depends_on`, which the table, TSV, and CSV do not show. `--json`, `--tsv`, and `--csv` exclude each other. `--status`, `--priority`, and `--specs-dir` do not apply to `--cross-branch`.

A missing or empty value for `--priority` or `--status` fails. Priority accepts `0` and other nonnegative integers outside the stored range of 1 through 3. These values return no matching assertions.

The two kinds of failure print on different streams. A bad filter value prints as JSON on stdout, so a caller that asked for `--json` reads the error in the format it expects. A malformed command line prints as text on stderr, as the mutually exclusive format flags do.

Every empty JSON result contains `type: "assertions"` and `assertions: []`, including an empty specs directory and filters with no matches. TSV and CSV contain their header. Table output reports no matches and includes each active filter value.

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
- A spec directory that has assertion files but no main spec file. The parser drops that directory and every assertion in it, so they are out of the queue.
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

A malformed field fails the parse of every file in the tree, not its own file alone. One bad line on the default branch stops every command that rebuilds the index. Run `spekk validate` before you commit an edit to `specs/`. See [Validation in CI and pre-commit](ci.md).

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
| `documents` | `kind` (`spec`, `assertion`, `doc`, or `raw`), `id`, `path`, `text_path`, `title`, `type`, `created`, `summary`, `sha256`, `state`. See [`spekk docs`](#spekk-docs) |
| `document_tags` | `kind`, `doc_id`, `tag` |
| `passages` | FTS5 table: `heading`, `body`, `terms`, and unindexed `kind`, `doc_id`, `start_line`, `end_line`, `is_header` |

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

## `spekk docs`

Search the specs, the docs in the repository, and the source documents in `raw/`, from one full-text index. See [Search Specs, Docs, and Source Documents](source-documents.md).

```bash
spekk docs search "torque limit" --kind raw,doc
spekk docs show pump-manual --lines 120:160
spekk docs add ~/Downloads/pump-manual.pdf
spekk docs add raw/pump-manual.pdf --summary "Acme P-200 pump manual: torque limits, valve sizes, and wiring."
spekk docs status
```

### `spekk docs search "<words>"`

Rank passages with BM25 and print the best passage of each document: kind, id, title, path, line range, summary, and the numbered lines of the passage that hold the most query words, so the answer is often in the search output. Up to three other matching passages of the same document are listed by line range. The output is at most 4,000 bytes, and results that do not fit are counted, not printed.

| Flag | Description |
|------|-------------|
| `--kind <k1,k2>` | Only these kinds: `spec`, `assertion`, `doc`, `raw` |
| `--type <type>` | Only documents with this frontmatter `type` |
| `--tag <tag>` | Only documents with this tag |
| `--since <YYYY-MM-DD>` | Only documents with a `created` date on or after this date |
| `--limit <n>` | At most n documents (default 5, maximum 20) |
| `--json` | One JSON object: `query`, `notice`, `results`, `omitted` |

Each word matches as written, joined with OR. `snake_case`, `dotted.name`, and `kebab-case` match as exact phrases, and `"quoted words"` match as one phrase. No query text is read as FTS5 syntax. An unknown kind, a malformed date, or a bad limit exits 2.

### `spekk docs show <id|path>`

Print numbered lines of one document, at most 200 for each call. The output names the next range.

| Flag | Description |
|------|-------------|
| `--lines <a:b>` | Lines a to b; `a:` prints from a |
| `--kind <kind>` | Choose the kind when two documents share the id |

Line numbers are those of the plaintext file: the document itself, or the derived text `formatted/<id>.txt` of a converted format. Text of kinds `doc` and `raw` is printed between markers that say it is untrusted data.

### `spekk docs add <file|folder>...`

Copy each file to `raw/` (a file already under `raw/` stays), convert it to text when it must, and write its stub in `formatted/`. Then rebuild the index.

| Flag | Description |
|------|-------------|
| `--summary "<text>"` | The summary, 40 to 600 characters. One file only |
| `--type <type>` | Document type, such as `manual`, `transcript`, `email` |
| `--tags <a,b>` | Tags |
| `--source <s>` | Where the document came from, such as a URL. The default is the path in the repository, or the file name for a file from outside |
| `--allow-secrets` | Add a file that the secret scan refused, after a person checked it |

Formats: `.md`, `.markdown`, `.txt`, `.html`, `.htm`, `.eml`, and `.xlsx` in the binary; `.pdf` with `pdftotext` and `.docx` with `pandoc` on `PATH`. A missing converter fails that file with the install command. Adding a document again with changed content updates the stub and clears the summary. A converted file larger than 5 MB is listed in `raw/.gitignore`. Exit 1 when any file failed.

### `spekk docs status`

List each source document that is not `ok` (`stale`, `no_summary`, `missing_raw`, `pointer`, `no_stub`) with the command to run. Exit 0.

## `spekk show`

Open the spec explorer, a web page that shows the spec tree.

On a screen that is 768 pixels wide or less, the page shows each spec as a full-screen card in a deck that snaps one card into view at a time. Swipe up on a card to see its assertions, and tap an assertion to read it. The search box, the hide-completed toggle, and the branch filter apply to the deck. The cross-branch metro map is on the wide layout only.

```bash
spekk show                                          # Write the page and open it
spekk show --watch                                  # Serve it and reload on change
spekk show --cross-branch                           # Merge preview
spekk show --cross-branch --branch-filter 'feat/*'  # Only branches that match the glob
spekk show -w --cross-branch                        # Merge preview with live reload
spekk show --json | jq                              # Print the data of the page as JSON
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--watch` | `-w` | Serve the page on `127.0.0.1` and reload it when the content of a spec file changes |
| `--cross-branch` | | Read-only preview of what a merge of each other branch would do to the specs |
| `--branch-filter <glob>` | | In cross-branch mode, only branches that match the glob (for example `'feat/*'`) |
| `--json` | | Print the data of the page to stdout as one JSON object. Write no file and open no browser. Not with `--watch` |
| `--help` | `-h` | Print the help for `spekk show` and exit |

Without `--watch`, `spekk show` writes `.spekk/index.html` and opens it in your browser. With `--watch`, it serves the page from a local port and reloads it when the content of a spec file changes. A save or `touch` that does not change the content does not reload the page. Set the `CI` environment variable to stop watch mode from opening a browser.

The explorer lists specs and assertions in the same order as `spekk list`: priority first, then ID. An item with no priority (0) comes last. In cross-branch mode, the items from other branches take their place in that order.

The page shows:

- The spec and assertion tree, with status and priority badges
- A metro map of the dependency chains
- Drag to pan, click to open a spec or an assertion
- Done specs hidden by default, with a toggle to show them
- A search box that filters by name, status, or priority

### JSON output

`spekk show --json` prints the data that the page uses, as one JSON object, and stops. It writes nothing to `.spekk/` and opens no browser. Use it when a script or a server needs the spec tree.

```json
{"projectName":"my-project","specs":[{"id":"auth","title":"Auth","status":"in_progress","priority":1,"file":"specs/auth/auth.md","content":"...","branch":"main"}],"assertions":[{"id":"login-form","parent":"auth","title":"Login form","status":"done","priority":1,"file":"specs/auth/assertions/login-form.md","content":"...","branch":"main","created":"2026-01-01T00:00:00Z"}]}
```

- `projectName` is the name of the project directory.
- `specs` and `assertions` are in the order of the page: priority, then ID, then file path.
- A `specs/` directory with no specs gives empty arrays and exit code 0.
- A parse error, for example a duplicate spec ID, gives exit code 1 and the error on stderr. Parser warnings also go to stderr, so stdout has only the JSON.
- With `--cross-branch`, the object also has `crossBranch`, and it can have `degraded` and `branches`. Each item can have `foreign` and `crossBranch` (a list of `branch`, `state`, `degraded`, `oldStatus`, `newStatus`), and each spec can have `crossBranchSummary`. `--branch-filter` applies.
- A cross-branch key with no value is absent from the output. An absent `degraded` or `foreign` means false, and an absent `branches` or `crossBranch` list means an empty list.

`--json --cross-branch` also works on a read-only bare repository. Set `GIT_DIR` to a directory that holds `HEAD` and a `commondir` file with the path of the bare repository, `GIT_WORK_TREE` to a directory that holds `specs/`, and `GIT_OBJECT_DIRECTORY` to an empty directory of your own, with `GIT_ALTERNATE_OBJECT_DIRECTORIES` set to the `objects` directory of the bare repository. The conflict check writes its objects to your directory, not to the bare repository.

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
spekk builder review                 # Review what was built, in a fresh session
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
spekk coach property-tests           # Write a property-based test for a promise
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
| `--mcp-config <file>` | Load the MCP servers in `<file>` for this run only, for example a browser that one skill needs. The observer gives `claude` the flag as `--mcp-config=<file>`. A missing value, or a path that is not a file, stops the run before `claude` starts |

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

The command refuses a `--type` outside the two values above, and refuses a stray argument. Both fail before it answers, because a `clear` result followed by an observation the parser drops on read loses the finding with nothing reported. A path list written with spaces instead of commas is the common case.

A malformed `.spekk/dont-flag.yaml` fails the check with a message that names the entry. A broken suppression file is never read as empty. See [Suppressing observations](configuration.md#suppressing-observations) for the file format.

### `spekk observer announce`

Announce the open observations on the connected chat surface. This command runs inside a sandbox session only, because it delivers through the session's conversation spool.

```bash
spekk observer announce
```

One run: `git fetch`, the only remote read, then refresh the index and pick the unannounced open observations with severity high or medium. Low never announces. An observation is eligible only as a live claim: it must sit on the `observer/<slug>` branch named after it, and no observation with that slug may be on main. This is the rule the digest and `scan-check` apply, so the three agree. Ordering is high severity first, then oldest first, then slug. It opens one conversation with at most three findings, then commits an `announced:` timestamp to each observer branch and pushes. An observation with no `affected` path never announces. With nothing to announce it prints `nothing to announce` and exits 0.

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

!!! note "Registration on the control host"

    With `SPEKK_API_TOKEN` set, `create` registers the agent on the control host through its API, and `destroy` revokes it. Without it, one manual step remains after `create`: register the agent token that the command prints on the control host, or run [`spekk sandbox register`](#spekk-sandbox-register-name) later. Until the agent is registered, it tries to connect and fails authentication.

    The control host receives only your spekk API token and the agent token that it makes itself. Your cloud and model credentials go from your machine to your sandbox, and nowhere else.

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
spekk sandbox create --name tmp-fix-login --one-off --purpose "fix the login test"
spekk sandbox create --name tmp-migration --one-off --expires 3d
spekk sandbox create --name my-sandbox --image base
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--name <name>` | Sandbox name (required). Lowercase letters, digits, and hyphens, starting with a letter or a digit |
| `--provider <provider>` | `digitalocean` (default), `gcp` for a Google Cloud VM, or `none` for a machine you already have. `--ip`, `--ssh-key`, or `--ssh-user` implies `none` |
| `--auth <mode>` | How the agent authenticates Claude: `bedrock` (default) or `subscription` |
| `--model <name>` | The model pin, `ANTHROPIC_MODEL` in the env file. Default: `ANTHROPIC_MODEL` from your environment, else Opus 5.5 in the form of the mode: `us.anthropic.claude-opus-5-5` for `bedrock`, `claude-opus-5-5` for `subscription`. `--model ""` writes no pin. See [A model pin belongs to its mode](configuration.md#the-control-host-sees-one-credential-of-yours) |
| `--region <region>` | DigitalOcean region (default: `nyc1`). For `gcp`, the zone |
| `--size <size>` | Droplet size slug (default: `s-2vcpu-4gb`). For `gcp`, the machine type |
| `--project <project>` | Assign the droplet to a DigitalOcean project, by name or UUID. For `gcp`, the project ID |
| `--vpc <uuid>` | Place the droplet in a DigitalOcean VPC. For `gcp`, the network name |
| `--provision-timeout <duration>` | How long to wait for cloud-init, as a Go duration (default: `30m`) |
| `--ip <address>` | Address of a machine you already have |
| `--ssh-key <path>` | Private key that reaches that machine as the login user |
| `--ssh-user <user>` | SSH login user for that machine (default: `root`). A non-root user must have passwordless sudo |
| `--release <tag>` | Pull the agent binary from a specific spekk release instead of the latest, e.g. an `exp-*` prerelease. The cloud-init template always comes from the installed CLI |
| `--organization <id>` | The control host organization to register the agent in. Needed only when you belong to more than one |
| `--image <name>` | Create the machine from a template image that `spekk sandbox image build` made. DigitalOcean and GCP. See [`spekk sandbox image`](#spekk-sandbox-image) |
| `--one-off` | Mark the sandbox one-off: one task, then destroy it. It expires 48 hours after `create` unless `--expires` says when. Without it, the sandbox is persistent and has no expiry |
| `--expires <when>` | Expiry of a one-off sandbox: a Go duration (`36h`), days (`3d`), a date (`2026-10-17`, which means 00:00 UTC), or an RFC 3339 time. It must be in the future, and it needs `--one-off` |
| `--purpose <text>` | Why the sandbox exists. It is kept in the record and in the marker on the machine. It is one line: `create` refuses a newline, a tab or any other control character |

The cloud flags and the existing-machine flags exclude each other. For the `gcp` defaults and what spekk creates in the project, see [A Sandbox on Google Cloud](advanced/sandbox-gcp.md). `create` refuses a name that is already recorded: destroy the old sandbox first.

**A DigitalOcean sandbox.** `create` checks that the environment variables for the auth mode are set, generates an SSH key pair under spekk's config directory, creates the droplet with the cloud-init template that ships in the CLI, and records it. The template installs Claude Code as `agent` with the native installer, from the `stable` release. It also installs the pinned Codex CLI and code mode host. A `gcp` sandbox follows the same steps on a Compute Engine VM. It then waits for SSH and for cloud-init to write `/opt/spekk/.provisioned`. The marker requires a successful Codex install. While it waits, it prints a progress line at most once a minute with the time waited and the last line of `/var/log/cloud-init-output.log`. When `cloud-init status --long` reports `error`, or a list of errors that is not empty, the wait stops at once and prints that status. When cloud-init reports `done` without the marker, the wait polls for 30 more seconds, because the last `runcmd` step can write the marker just after cloud-init reports `done`. When the marker appears, `create` injects the credentials, configures git for the agent, puts the agent binary and the spekk CLI for the machine's CPU architecture from the release of this spekk on the machine (see [`deploy`](#spekk-sandbox-deploy-name)) and deploys them, marks the sandbox `active`, and prints the result.

**Registration.** With `SPEKK_API_TOKEN` set, `create` registers the agent on the control host (`POST /api/agents/`) after it fetches the release artifacts and before it asks the provider for a machine. A refused registration costs nothing. The token that the control host returns is the `SPEKK_AGENT_TOKEN` that `create` writes to the machine, and the record stores the agent id. When a later step fails, or when you press Ctrl-C (`SIGINT`) or the process gets `SIGTERM`, `create` revokes the agent again and says so. If that revoke also fails, the record keeps the agent id, so `destroy` can revoke it. On success, `create` prints the agent id and not the token. Without `SPEKK_API_TOKEN`, `create` makes the token itself, prints it to register by hand, and tells you how to run `spekk sandbox register` later. When you belong to more than one organization and give no `--organization`, the control host refuses the registration, and the error tells you to pass `--organization <id>`. The id is in the `organization` field of an agent you already have (`GET /api/agents/` on the control host), or an admin of the organization can give it to you.

When the wait runs out, the droplet keeps running and the record stays at `provisioning`. Nothing is destroyed. Finish it with [`spekk sandbox provision`](#spekk-sandbox-provision-name) when cloud-init is done, or remove it with `destroy`.

**A machine you already have.** spekk does not provision a machine it did not create. Prepare the machine yourself, so that it carries `/opt/spekk/.provisioned`, an `agent` user, and the directories the agent needs. `create` checks the marker over SSH, then runs the same three steps: inject the credentials, configure git, deploy the agent. With `--ssh-user`, the three steps and the later teardown run under `sudo`. Everything else runs as the login user.

To prepare a Debian/Ubuntu machine (amd64 or arm64, including a Raspberry Pi), copy `prepare-machine.sh`, `configure-agent-attribution.py`, and `assisted-by` from the same checkout to it and run the setup as root:

```bash
scp scripts/prepare-machine.sh scripts/configure-agent-attribution.py scripts/assisted-by user@your-machine:
ssh user@your-machine 'sudo bash prepare-machine.sh'
```

It installs the `agent` user, Docker, the Claude Code CLI (the native binary from the `stable` release, installed as `agent` and linked to `/usr/local/bin/claude`, no Node runtime), the pinned Codex CLI and code mode host, the spekk CLI, `git`/`gh`, and the spekk directories, then writes the marker - the same setup a droplet gets from cloud-init, minus the droplet-only hardening (a full package upgrade, a default-deny firewall, fail2ban) that could lock you out of a machine you already use. A later run repairs a missing, damaged, or different-release Codex host. It picks Docker's Debian or Ubuntu repository from the host, so it works on a Debian-based Raspberry Pi as well as an Ubuntu server. To provision by hand instead, its steps are the checklist to follow.

The agent owns `/opt/spekk/workspace`, so on a machine you also use interactively your login user cannot edit what the agent writes. Pass `SPEKK_SHARE_USER=<login>` (e.g. `sudo SPEKK_SHARE_USER=pi bash prepare-machine.sh`) to grant that user read/write on the workspace with ACLs. ACLs, not group ownership: `deploy` re-runs `chown -R agent:agent /opt/spekk` on every deploy, which preserves ACLs but would wipe a group change, and the default ACL also covers the files the agent creates later.

The agent binary spekk deploys matches the machine's CPU architecture, so an arm64 host (a Raspberry Pi, an arm64 droplet) is served an arm64 build. spekk pulls that build from the release of this spekk by default (the latest release for a dev build); a release only carries the architectures it published. To use a build that only exists in a prerelease, pin it with `--release <tag>` (e.g. `--release exp-arm64`).

**Setup recipes.** Each recipe pairs a provider with an auth mode. Export the variables the mode needs first ([Sandbox provisioning](configuration.md#sandbox-provisioning) lists them all), then run one command.

DigitalOcean droplet, Bedrock (both defaults):

```bash
export DO_API_TOKEN=... SPEKK_HOST=... GITHUB_TOKEN=...
export AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=... AWS_DEFAULT_REGION=us-east-1
spekk sandbox create --name my-sandbox
```

DigitalOcean droplet, Claude subscription:

```bash
export DO_API_TOKEN=... SPEKK_HOST=... GITHUB_TOKEN=...
export CLAUDE_CODE_OAUTH_TOKEN="$(cat ~/.config/spekk/oauth-token)"
export CODEX_AUTH_JSON_FILE="$HOME/.config/spekk/codex-auth.json"
spekk sandbox create --name my-sandbox --auth subscription
```

Subscription mode requires the Claude token. `CODEX_AUTH_JSON_FILE` is optional. If set, it must be a regular file owned by the current user with no group or other access. Create it with `codex login` using `cli_auth_credentials_store = "file"`. The CLI checks for a ChatGPT refresh token, sends the cache over SSH stdin, and replaces `/home/agent/.codex/auth.json` as an agent-owned 0600 file. It refuses unsafe existing paths. One ChatGPT login supports one active Codex machine at a time. If a caller seeds several machines from one file, the second refresh invalidates the other copies. The CLI does not coordinate machines or copy rotated tokens back; callers must serialize use and copy back the latest cache.

A machine you already have (`--provider none`), Bedrock:

```bash
export SPEKK_HOST=... GITHUB_TOKEN=...
export AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=... AWS_DEFAULT_REGION=us-east-1
spekk sandbox create --name my-box --ip 203.0.113.10 --ssh-key ~/.ssh/my-box --ssh-user ubuntu
```

`--ip` implies `--provider none`, so no DigitalOcean token is needed. For a subscription machine, swap in `--auth subscription` and export `CLAUDE_CODE_OAUTH_TOKEN` in place of the AWS keys. To change any of these after the sandbox exists, see [Updating sandbox variables](configuration.md#updating-sandbox-variables).

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
| `--release <tag>` | | Pull the agent binary from a specific spekk release instead of the latest, e.g. an `exp-*` prerelease |
| `--force` | `-f` | Provision a sandbox whose status is not `provisioning` |
| `--organization <id>` | | The control host organization, when `provision` registers a new agent |

`provision` checks that the environment variables for the auth mode are set, checks that `/opt/spekk/.provisioned` exists on the machine, and then runs the same three steps `create` runs after its wait, in the same order and through the same code. It marks the sandbox `active`. A sandbox recorded before 1.28.0 has no auth mode on record, and reads as `bedrock`. The env file gets the model pin of `create`: `ANTHROPIC_MODEL` from your environment, else the default of the mode. `provision` has no `--model` flag.

The agent token follows the record:

- **A registered sandbox** (the record has an agent id) keeps its agent. `provision` reads `SPEKK_AGENT_TOKEN` from `/etc/spekk/agent.env` on the machine and writes the same token back, so the agent keeps its connection, its schedules, and its history. If it cannot read a token, it stops before it changes the machine, and tells you to run `spekk sandbox register`. The error shows the SSH failure, or says that the file has no token.
- **A sandbox with no agent** gets a new token. With `SPEKK_API_TOKEN` set, `provision` registers it as `create` does. If the record cannot be saved after that, the error names the new agent id. Without it, `provision` prints the token to register by hand.

`provision` is also how you change a live sandbox's credentials from your machine: re-run it with the variables set in your environment to switch auth mode, rotate a token, or push a new AWS key. See [Updating sandbox variables](configuration.md#updating-sandbox-variables) for that and for the on-machine alternative.

### `spekk sandbox register <name>`

Register a recorded sandbox on the control host with your API token.

```bash
spekk sandbox register my-sandbox
spekk sandbox register my-sandbox --organization <id>
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--organization <id>` | The control host organization to register the agent in. Needed only when you belong to more than one |

`register` needs `SPEKK_API_TOKEN` and `SPEKK_HOST`. It registers a new agent, then, over SSH, replaces the `SPEKK_AGENT_TOKEN` and `SPEKK_AGENT_ID` lines in `/etc/spekk/agent.env`, keeps every other line, and restarts `spekk-agent`. The script keeps a copy of the old file as `agent.env.bak` until the restart succeeds. If a step fails, it puts the old file back and restarts the agent with the old token. Then `register` revokes the new agent, and the record keeps the old agent id, so the machine and the record agree. On success, the record stores the new agent id. If the record named an agent before, `register` revokes that old agent, because its token is no longer on the machine.

Use it for a sandbox that was created without `SPEKK_API_TOKEN`, or to replace the agent of a sandbox. A new agent starts with no schedules.

### `spekk sandbox list`

Print every recorded sandbox, sorted by name: name, kind, IP, region, status, creation time, and expiry. A sandbox whose expiry is in the past shows `EXPIRED`. A record that an older CLI wrote has no kind, and lists as `persistent`.

```bash
spekk sandbox list
spekk sandbox list --expired             # Only the expired sandboxes
spekk sandbox list --all-hosts           # Also the labeled sandboxes that other machines created
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--expired` | Show only the sandboxes whose expiry is in the past |
| `--all-hosts` | Also ask DigitalOcean and GCP for the labeled machines that no local record names, and show them with the status `remote` |

`--all-hosts` lists the DigitalOcean droplets with the `spekk-sandbox` tag, and the GCP VMs with a `spekk-kind` label in the default gcloud project. For each cloud that it cannot ask (no token, no `gcloud`, or an API error), it prints one line on stderr with the reason. Expiry is a reminder only: spekk never destroys a sandbox because it expired. See [Sandbox kinds](advanced/sandbox-architecture.md#sandbox-kinds).

### `spekk sandbox status <name> [--json]`

Print the stored fields of a sandbox (with its kind, purpose, expiry and the host that created it), the live machine state, and five SSH checks: the provisioned marker, the `spekk-agent` service, the handshake record of the agent, and the version of the spekk CLI and of the agent binary on the machine. The two version commands run as `agent`, which owns both binaries, and not as the login user.

```bash
spekk sandbox status my-sandbox
spekk sandbox status my-sandbox --json
```

`--json` prints one JSON object with the same values as the text lines, with snake_case keys (for example `agent_service` and `secret_key_fingerprint`). A version warning is in its own key, for example `agent_client_warning`. Warnings about the provider stay on stderr.

When the cloud API cannot be reached, or the provider cannot be built because its token is missing, `status` prints a warning and shows the stored status marked `(stored)`. For a machine spekk did not create there is no live state, and the stored status is shown with no warning.

A version line gets `WARNING:` and the command that fixes it when the version is not the release that `spekk sandbox deploy <name>` installs (the release of this spekk, or the latest release for a dev build), when it is `dev` (a binary with no release stamp), or when it is `missing`. For example:

```
spekk CLI: v1.32.0  WARNING: not v1.38.1, the release of this spekk; update with: spekk sandbox deploy my-sandbox
Agent client: v1.38.1
```

Nothing updates a sandbox on its own. A sandbox runs real work, so the update is the one command, `deploy`.

The service can be `active` while the control host refuses every dial of the agent, for example with HTTP 403 for a token that is not valid. The agent records its last successful handshake and its last failed dial in `/var/log/spekk/agent-connection.json`, and `status` shows both. When the last dial failed, or the agent never connected, the failure line gets a warning. For example:

```
Agent last connected: never
Agent last handshake failure: 2026-10-23T05:10:00.123Z (HTTP 403)  WARNING: the agent is not connected to the control host
```

An agent binary older than this check writes no record, and the lines then say `unknown (no record; ...)`.

The `Secret key fingerprint` line shows the fingerprint of the key that the agent uses to receive secrets from the control host: the first 16 hex characters of SHA-256 of its X25519 public key. The agent keeps the private key in `/home/agent/.config/spekk-agent/secret-key` (mode 0600) and never sends or logs it. The Secrets section of the control host shows the fingerprint of the key that the sandbox sent. An organization admin compares the two values before accepting the key. For example:

```
Secret key fingerprint: 300c9c9603b92a4b
```

When the agent refused its key file (a file that other users can read, a symbolic link, or contents that are not a key), the line says `none (<reason>)`. The agent does not replace the file. Fix or remove it, then restart the agent: a new key needs a new accept in the Secrets section. An agent binary older than this check says `unknown (no fingerprint in the record; ...)`.

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
spekk sandbox destroy my-sandbox --ignore-revoke-failure   # Destroy even if the agent cannot be revoked
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--force` | `-f` | Skip the confirmation prompt, and remove the record when the teardown cannot run |
| `--ignore-revoke-failure` | | Destroy the sandbox when its agent cannot be revoked on the control host |

When the record has an agent id, `destroy` first revokes the agent on the control host (`POST /api/agents/<id>/revoke/`), before it touches the machine. This needs `SPEKK_API_TOKEN` and `SPEKK_HOST`: without them, `destroy` stops before the confirmation prompt and says so. A failed revoke stops `destroy`, and the machine and the record stay. `--force` does not change this. With `--ignore-revoke-failure`, `destroy` prints a warning that names the agent id and continues; revoke that agent on the control host.

For a DigitalOcean sandbox, `destroy` deletes the droplet and its SSH key on DigitalOcean, then removes the local record. When the record names no droplet, `destroy` refuses, because the droplet may still be running and billing. `--force` removes the record anyway.

For a machine spekk did not create, `destroy` stops the `spekk-agent` service, removes the credential files spekk put on the machine and the marker `/opt/spekk/sandbox.json`, and removes the local record. It makes no cloud API call and needs no API token. When the teardown fails, `destroy` keeps the record, because removing it would leave an agent running with live credentials and nothing that points at it. `--force` removes the record anyway.

The local SSH key pair is deleted only when spekk generated it. The test is where the private key is: the recorded path, cleaned and made absolute, must be inside the `keys` directory under spekk's config directory, and so must the file it resolves to after symlinks are followed on both sides. A key anywhere else is kept, and `destroy` prints one line to stderr that names the path it kept.

### `spekk sandbox deploy <name>`

Download the agent binary and the spekk CLI of one release and install them on a sandbox.

```bash
spekk sandbox deploy my-sandbox
spekk sandbox deploy my-sandbox --release v1.38.1
```

Without `--release`, a CLI with the version `vX.Y.Z` installs the release `vX.Y.Z`, so the sandbox matches the CLI that manages it. Any other build (`dev`) installs the latest release, and says so. `create` and `provision` pick the release the same way.

Before the files go to the machine, `deploy` reads the SHA-256 that GitHub publishes for each release asset (`digest` in `GET /repos/spekk-ai/spekk-cli/releases/tags/<tag>`, an anonymous API call). An asset with no digest stops the deploy.

The machine downloads both files from the release itself (`https://github.com/spekk-ai/spekk-cli/releases/download/<tag>/`), so no binary crosses the connection of your computer. The download runs over SSH as the login user, with `curl`, or with `wget` when `curl` is not installed, and `sha256sum -c` checks each file against the release digest. A failure or a mismatch removes both downloaded files. `curl` stops an attempt that is below 1024 bytes per second for 30 seconds or that runs for more than 300 seconds, and it tries 3 times (`--retry 2`). Thus a slow download can take approximately 15 minutes for each file before the fallback starts, and approximately 30 minutes for the two files. `wget` runs under `timeout 300`, which is the limit for each file. When the download on the machine fails for any cause (no `curl` or `wget`, no `timeout` command, a `wget` that cannot complete the download, such as a BusyBox `wget` that fails the TLS connection, no route to GitHub, a mismatch, a time limit), `deploy` prints `spekk: the machine could not download release <tag> (<cause>); copying it from this computer`. Then it downloads both files to your computer and copies them to the machine with `scp`, as before 1.49.0.

Either way, on the machine `sha256sum -c` checks each staged file again before anything is replaced: a file that does not match stops the deploy with `spekk: <file> does not match the release sha256`, and the installed binaries and the running service stay as they were.

The agent goes to `/opt/spekk/agent-client`. The CLI goes to `/opt/spekk/cli/spekk`, owned by `agent`, so `spekk update` as `agent` can replace it, and `/usr/local/bin/spekk` links to it. When `~agent/.local/bin/spekk` is there (an earlier install put a copy there), it becomes a link to the same file. `agent` owns `/opt/spekk` and its home, so `deploy` does not follow a link there: a link at `/opt/spekk/cli` is replaced with a directory, and when `~agent/.local` or `~agent/.local/bin` is a link, `deploy` does not change `spekk` in it and says so. Then `deploy` writes the `spekk-agent` systemd unit and restarts the service. Every login stages the files in its home directory, then installs each beside its target and renames it into place, so a deploy replaces an agent that is running.

Before the restart, it reads the version of the `claude` that the service runs, with the service's user and `agent.env`. When the version is older than the minimum (`MinVersion` in `internal/claudecode`), or cannot be read, it installs the `stable` release as `agent` with the native installer, and reads the version again. When the version is still older, the deploy fails and does not restart the agent, because the new agent refuses every turn on an older Claude Code. When the version still cannot be read, the deploy prints a warning and continues. A `claude` process that runs keeps its own binary, so this step stops no turn. `create` and `provision` run the same step when they deploy the agent.

### `spekk sandbox image`

Build, list, and delete template images. A sandbox made from an image skips the package installation of cloud-init, so a create takes about a minute instead of about four.

```bash
spekk sandbox image build --name base
spekk sandbox image build --name web-app --setup infrastructure/sandbox/image-setup.sh
spekk sandbox image build --name base --provider gcp --regions us-east1-b
spekk sandbox image list
spekk sandbox image delete web-app
```

**`image build` flags:**

| Flag | Description |
|------|-------------|
| `--name <image>` | Image name (required): lowercase letters, digits, and hyphens, starting with a letter, at most 63 characters |
| `--setup <script>` | A script that runs as root on the builder before the image is made. A project keeps its own script in its own repository |
| `--provider <provider>` | `digitalocean` (default) or `gcp` |
| `--size <size>` | Size of the builder (default: the provider default) |
| `--regions <r1,r2>` | DigitalOcean: the builder region, then the regions that the snapshot is copied to (default: `nyc1,nyc3`). GCP: the zone of the builder. A GCP image is global |
| `--project <id>` | GCP project (default: the gcloud project) |

**Build steps.** The build creates a temporary builder sandbox, `image-<name>`, with the normal cloud-init, and records it in `sandboxes.json` with the status `building`. It registers no agent and injects no credential. It runs `--setup` as root, with its output on the terminal. Then one SSH session cleans the builder and checks it (see below). Then the provider makes a private image: a snapshot of the shut-down droplet, copied to each region of `--regions`, or a custom image of the boot disk of the stopped GCP VM. The build records the image in `images.json` beside `sandboxes.json`: name, provider, provider image ID, regions, GCP project, created date, spekk version, OS version, and the SHA-256 of the setup script.

The builder is destroyed on success, on each failure, and on Ctrl-C or SIGTERM, with its record, its key pair, and its known_hosts file, and the build prints what it removed. When the destroy fails, the build prints the `spekk sandbox destroy image-<name> --force` command that removes it. When the provider made the image but a later step failed (for example, a copy to a region), the record is still written with the regions that have the image, so `image delete` can remove it.

**Image hygiene.** "Each home" is `/root`, each directory in `/home`, and the home of each user in `/etc/passwd`. The clean step removes what makes each machine its own, and the logs: the SSH host keys, the content of `/etc/machine-id`, `/opt/spekk/.provisioned`, the cloud-init state (`cloud-init clean --logs`), the content of the files in `/var/log`, and in each home the authorized keys, each `.*_history` file, `.viminfo`, the recipient key directory of the agent client (`.config/spekk-agent`), and the Claude Code settings (`.claude`, `.claude.json`) when they hold no sign-in. The clean step does not remove credentials. The check fails the build, with one line for each finding, when the builder has:

- `/etc/spekk/agent.env`
- in a home: a file in `.ssh` (also when `.ssh` is a symlink) other than `known_hosts`, `config`, or a `*.pub` file; a Claude Code sign-in (`.claude/.credentials.json`) or `.claude.json`; `.netrc`, `.aws/credentials`, `.git-credentials`, `.config/git/credentials`, `.config/gcloud`, a DigitalOcean CLI login (`.config/doctl/config.yaml`), or a GitHub CLI login (`.config/gh/hosts.yml`)
- in a text file in a home, `/opt`, `/srv`, or `/var/lib` (but not `/var/lib/docker` or `/var/lib/containerd`): a private key (a `-----BEGIN ... PRIVATE KEY-----` line, or the same header followed by `\n` in a JSON string). The directories `node_modules`, `site-packages`, `dist-packages`, and `archive-v0` (the unpacked wheels in the uv cache) are not searched for keys, because installed packages often ship test keys
- in a text file in the same directories or in `/etc`: a token in a known format (GitHub, Anthropic, AWS, DigitalOcean, GitLab, or Slack), as a full word in its real length. A match inside a longer string (for example in base64) or a short test value such as `xoxb-test-token` is not a finding
- a git config (a `*.git/config` file in the same directories, `.gitconfig` or `.config/git/config` in a home, or `/etc/gitconfig`) with a user or token in an `http(s)://` URL, or with an `extraheader` line
- a Docker registry login, an npm token, or a line that sets a variable whose name has `TOKEN`, `SECRET`, `API_KEY`, `PASSWORD`, or `ACCESS_KEY`, in `/etc/environment`, `/etc/bash.bashrc`, `/etc/profile.d/`, a bash or zsh startup file in a home, a `.env` file in the same directories, or an `Environment=` line in `/etc/systemd/system`

It also fails when a step of the clean step did not take effect. The check is best effort: it fails on known secret shapes and locations, but it cannot find every secret, and the author of the setup script stays responsible for what the image contains. A setup script that needs a credential must remove it before it ends. The setup script gets `/dev/null` as stdin, so a step that reads stdin does not skip the steps after it.

**Create from an image.** `spekk sandbox create --image <name>` finds the image in `images.json`, or, when there is no record, by name in the provider's own image list. That list has only the images that `image build` made: on DigitalOcean, the snapshots named `spekk-image-<name>`, and on GCP, the images with the `spekk-image` label. The machine starts from the image with a short first-boot cloud-init: the `agent` user with the sandbox key, then `/opt/spekk/.provisioned`. Cloud-init makes new SSH host keys and systemd makes a new `/etc/machine-id`, so two sandboxes from one image do not share them. The marker check, the credential injection, and the agent deploy are the same as for a fresh machine. The deploy still upgrades a Claude Code that is older than the minimum, and the agent binary always comes from the release, never from the image. The sandbox record has `image: <name>`.

When the provider has no image with that name, or a DigitalOcean image has no copy in the region of the create, `create` prints one line and uses the full cloud-init:

```
No image "web-app" for digitalocean in sfo3; creating from the stock image with the full cloud-init.
```

`--image` with a machine you already have (`--ip`, `--provider none`) is an error. When the image is older than 14 days, `create` prints a warning with the age and continues.

**Rebuild, do not upgrade at boot.** Rebuild each image every week, for security updates, and with each spekk release: `image delete`, then `image build` with the same `--setup`. The first boot does not run `apt upgrade`, because that removes the time saving. Images are private to the account: a DigitalOcean snapshot and a GCP custom image are private by default, and the build does not share them.

**`image list`** prints one row for each record: name, provider, image ID, regions, created date, age in days, spekk version, and OS version. **`image delete <name>`** deletes the provider image (on DigitalOcean, every regional copy), then removes the record. When the provider no longer has the image, it prints a warning and removes the record.

**AWS.** spekk does not make AMIs. To start the CloudFormation sandbox from your own AMI, see [A Sandbox on AWS](advanced/sandbox-aws.md#start-from-your-own-ami).

### Machine mode (`--machine`)

Machine mode is the interface for a server worker that launches sandboxes for its users. A person at a terminal does not use it. The worker gives each secret to the CLI on stdin, and reads the progress from stdout. No secret is in a flag, in the environment of the CLI, or on stdout, and machine mode reads no credential from its environment. Machine mode is for DigitalOcean only.

```bash
spekk sandbox create <name> --machine --size <size> --region <region> [--one-off [--expires <when>]] [--label <key>=<value> ...] [--release <tag>] [--provision-timeout <duration>] < create.json
spekk sandbox destroy <name> --machine --force < destroy.json
```

**Create input.** One JSON object on stdin. Each field is required and must not be empty, except the optional GitHub pair and `image` (see below). A field that is not in this list, data after the object, or more than 64 KiB is an error:

```json
{
  "digitalocean_token": "<DigitalOcean token>",
  "registration_token": "<one-time registration token>",
  "spekk_host": "<control host>",
  "github_token": "<GitHub App installation token>",
  "github_repository": "<owner>/<repository>",
  "harness": "claude-code",
  "model_credential": {"kind": "none"}
}
```

Omit both `github_repository` and `github_token` to start without a repository. Both fields can also be empty. If either field has a value, both must have a non-empty value, and `github_repository` must be `<owner>/<repository>`. The CLI uses no GitHub token from its environment. A launch without a repository refuses an existing `.git` file, directory, or symbolic link in the resolved workspace or any parent directory. It reports the entry and preserves all files. This also applies to template images.

The object can also have `"image": "<name>"`, which is optional. It names a template image. The CLI finds the image by that name in the private images of the DigitalOcean account, because the worker has no `images.json`. Without the field, or when the region has no copy of the image, the droplet gets the full cloud-init. The stage lines are the same either way.

`harness` must be `claude-code`. `model_credential` is one of these kinds. Each kind takes only its own fields. A Claude subscription token is not a choice in machine mode, because it belongs to one person:

| `kind` | Fields | Env file on the machine |
|------|--------|-------------------------|
| `none` | none | No model credential and no model pin. A person signs in to Claude Code on the machine later. Until then the heartbeat of the agent says `claude_signed_in: false` (see [Claude sign-in state](advanced/sandbox-architecture.md#claude-sign-in-state)) |
| `anthropic_api_key` | `api_key` | `ANTHROPIC_API_KEY`, and the model pin `ANTHROPIC_MODEL=claude-opus-5-5` |
| `bedrock` | `aws_access_key_id`, `aws_secret_access_key`, `aws_region` | The AWS keys, `AWS_DEFAULT_REGION`, `CLAUDE_CODE_USE_BEDROCK=1`, and the model pin `ANTHROPIC_MODEL=us.anthropic.claude-opus-5-5` |

The input has no model field, so the pin is always the default of the kind.

**Flags.** `--size` and `--region` are required. `--label <key>=<value>` can repeat. The key and the value are lowercase `[a-z0-9_-]`, 1 to 63 characters, and the label goes on the droplet as the tag `<key>:<value>`, next to the kind tags of a terminal create. The label tags are in the droplet create request, so a droplet that a killed or failed create leaves has them too. A key that spekk sets itself (`spekk-kind`, `spekk-expires`, `spekk-host`, `spekk-sandbox`) is refused. Any other flag is an error. Without `--release`, a CLI with the version `vX.Y.Z` deploys the agent of the release `vX.Y.Z`, so the CLI and the agent match. Any other build needs `--release <tag>`.

**Stage lines.** Stdout has one JSON line for each stage, and nothing else. Human logs go to stderr:

```json
{"stage":"creating"}
{"stage":"provisioning","droplet_id":123,"ip":"203.0.113.10"}
{"stage":"registering"}
{"stage":"online","agent_id":"<id>"}
```

While the CLI waits (for the release download, the droplet, cloud-init, or an SSH step), it writes the last stage line again, unchanged, every 30 seconds. A line comes at least every 30 seconds, and the final `online` or `failed` line is always the last line.

On an error, the last line is `{"stage":"failed","at":"<stage>","message":"<text>"}`, where `at` is the last stage that started, and the exit status is 1. The message holds no secret. A bad flag or a bad input fails at `creating`, before any call to DigitalOcean, the control host, or GitHub.

**Steps.** `creating` checks the input and creates the droplet with the token from stdin. `provisioning` waits for cloud-init. `registering` makes the agent token, registers the agent with `POST /api/agents/register/` and `Authorization: Bearer <registration token>` (the body has `name`, `sandbox_id`, and `token`), writes the env file, prepares `/opt/spekk/workspace`, and deploys the agent. With a repository, it clones into that workspace and configures git. Without a repository, it creates a plain directory owned by `agent` and sets no Git credential helper or remote. Each script that holds a secret goes to the machine on the stdin of `ssh`.

**Git.** When a repository is supplied, the installation token reaches `git clone` in the environment of the git process, and is not stored. Then `credential.https://github.com.helper` is `/opt/spekk/agent-client git-credential`. For each git request to `github.com`, the helper reads `/etc/spekk/agent.env` and asks the control host for a new installation token with `POST /api/agents/<agent id>/git-token/` and `Authorization: Bearer <agent token>`. The env file has no `GITHUB_TOKEN`, and `gh` is not signed in.

**Failure.** The CLI does not destroy the droplet and does not revoke the agent when a create fails. It keeps the record of a droplet that exists, so the caller can run `destroy --machine` with the same `HOME`, and revoke the agent itself.

**State.** The record and the SSH key are under `$HOME/.config/spekk/` (or `$XDG_CONFIG_HOME/spekk/`), as for a terminal create. Give each launch its own `HOME`, and keep it until the destroy.

**Destroy.** `destroy --machine --force` reads `{"digitalocean_token": "<token>"}` on stdin, with the same strict rules. It needs `--force`, because stdin cannot also answer the prompt. It does not revoke the agent. It deletes the droplet, its DigitalOcean SSH key, the local key pair, the `known_hosts` file, and the record. It writes nothing to stdout, and exits 0 when the record is gone.

## `spekk schedule`

Manage the scheduled tasks of a sandbox agent on the control host. Every subcommand needs `SPEKK_API_TOKEN` and `SPEKK_HOST`.

```bash
spekk schedule list my-sandbox
spekk schedule create my-sandbox --slug nightly-scan --cron_expression "0 9 * * *" --task_text "Scan the specs for drift"
spekk schedule update my-sandbox nightly-scan --timezone America/New_York --enabled false
spekk schedule run my-sandbox nightly-scan
spekk schedule delete my-sandbox nightly-scan
```

`<agent>` is the name of a recorded sandbox that has an agent id, or an agent id. `<schedule>` is a schedule id or its slug.

| Subcommand | What it does |
|------------|--------------|
| `list <agent>` | Prints each schedule, from all pages: id, slug, cron expression, timezone, role, enabled, and next fire time |
| `create <agent> [fields]` | Creates a schedule and prints it as JSON. `--slug`, `--cron_expression`, and `--task_text` are required by the control host |
| `update <agent> <schedule> [fields]` | Changes only the given fields and prints the schedule as JSON |
| `delete <agent> <schedule>` | Deletes the schedule |
| `run <agent> <schedule>` | Runs the schedule now and prints the message id. When the agent is disabled or not connected, it prints the reason from the control host |

**Fields.** The flag names are the control host API field names.

| Flag | Description |
|------|-------------|
| `--slug <slug>` | Stable job name, unique for the agent |
| `--cron_expression <expr>` | Five-field cron expression, for example `"0 9 * * *"` |
| `--timezone <zone>` | IANA zone that the expression runs in (default: `UTC`) |
| `--task_text <text>` | The message the agent receives |
| `--custom_context <text>` | Context added to the session system prompt. `--custom_context ""` clears it |
| `--role <role>` | Session role (default: `builder`) |
| `--enabled true\|false` | A disabled schedule never fires |

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
| `--channel <name\|id>` | Slack channel joined by the agent's bot; accepts a name, `#name`, or ID. Omit to use the home channel |

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
spekk update                          # Install the latest release
spekk update --check                  # Print the current and latest version, install nothing
spekk update --version exp-arm64      # Install a specific tag, e.g. a prerelease
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--check` | `-c` | Report whether an update exists, and install nothing |
| `--version <tag>` | | Install a specific release tag instead of the latest. GitHub excludes prereleases from "latest", so this is how you get onto an `exp-*` build. It skips the newer-version check, so it can move to an older or a prerelease tag, and it works from a development build |

The update writes to the install directory, so that directory must be writable. See [Install](install.md#updating). A development build (`spekk version` prints `dev`) cannot update to the latest, but `--version <tag>` still works from one. After the check or the install, `spekk update` scans the files `spekk install --target` wrote and reports each one that no longer matches the binary, with the command to run.

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
