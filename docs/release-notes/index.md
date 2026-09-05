---
icon: lucide/megaphone
---

# Release Notes

What's new in each version of Spekk CLI.

---

## [1.28.0 -- A Slow cloud-init No Longer Costs You the Droplet](RELEASE-NOTES-1.28.0.md)

`spekk sandbox create` waited a fixed ten minutes for cloud-init, and a slow apt upgrade could take eighteen. When it gave up, the droplet kept running and the record stayed at `provisioning`, with no command to finish it. The wait is now `--provision-timeout` (default 30 minutes), it prints progress once a minute, and it stops early when cloud-init reports an error. `spekk sandbox provision <name>` finishes a sandbox the wait left behind.

## [1.27.0 -- A Sandbox That Does Not Admit Root](RELEASE-NOTES-1.27.0.md)

1.26.0 let a sandbox be a machine you already have, and assumed that machine lets you log in as root. Many do not: an AWS Ubuntu AMI gives you `ubuntu` and disables root over SSH. `--ssh-user <user>` logs in as that user and escalates the four steps that need root with `sudo`. A sandbox recorded before this release reads as root, so nothing changes for one that already exists.

## [1.26.0 -- A Sandbox Stops Being a Droplet](RELEASE-NOTES-1.26.0.md)

A sandbox could only be a DigitalOcean droplet that spekk created, billing Claude through Bedrock. This release makes the machine, the cloud that owns it, and the account that pays each a separate choice: a `Provider` interface with DigitalOcean as one implementation, `--ip`/`--ssh-key` to register a machine you already have, and `--auth subscription` to pay with a Claude subscription instead of the Bedrock API. Nothing changes for an existing sandbox -- Bedrock stays the default. **Earlier releases wrote a live `ANTHROPIC_API_KEY` into the agent's login shell, outside the env file and outside what `destroy` removed; this release stops writing it and clears it. Read the upgrading section.**

## [1.25.0 -- Four Silent Failures](RELEASE-NOTES-1.25.0.md)

Every fix here is for something that failed without saying so. A dropped WebSocket killed the turn it was carrying, so a long job reported nothing at all. A follow-up message for a running session wedged the sandbox permanently, while its heartbeats kept it looking healthy. An observation suppressed new findings from any branch, so drift a team had already fixed once could never be reported again. And a comment in frontmatter quietly changed a value or discarded a list. **A value with an unquoted `#` now loses its tail, which rewrites rows an existing index holds -- read the upgrading section before you take this.**

## [1.24.0 -- An Observation's Own Keys, Indexed](RELEASE-NOTES-1.24.0.md)

A custom frontmatter key on an observation validated, survived a round trip, and reached no table, so it was worse than prose: a reader assumes a key is queryable. Provenance an observation carries beyond the lifecycle set -- which skill found it, which run, which document narrated it -- was unreachable from `spekk query`. Such a key now reaches `frontmatter_fields` under `owner_type = 'observation'`, keyed on the slug, under the same split rule a spec and an assertion already use. `affected` stays out: it is the evidence gate and the dedup key, and `observation_files` is its table. The index schema version goes to 4, and existing databases rebuild transparently.

## [1.23.0 -- Quieter Output and a Real Branch Check](RELEASE-NOTES-1.23.0.md)

Two warnings made the output difficult to read. The branch guard compared a value with 14 fixed words and did not read git, so it accepted a branch that does not exist and refused team names that git accepts. This release deletes the guard, and `spekk validate` reads the refs instead. The parser now gives one line for the files that it skips, and `spekk next` no longer writes each warning two times. `validate` also stops making a `locked-by` value necessary on an `in_progress` assertion, which a coach cannot make, and it reports an old lock instead. Installs identify a managed file by its path, and `--project` uses the repository root. **`spekk validate` has three new failure conditions. Read the upgrading section before you change a pin.**

## [1.22.0 -- Validation Becomes a Gate](RELEASE-NOTES-1.22.0.md)

`spekk validate` was available before this release, but nothing made it run. A `spekk-validate` pre-commit hook now finds an incorrect field before the commit exists, and a CI gate finds what the hook does not: a new clone without `pre-commit install`, and `git commit --no-verify`. Each agent path that writes to `specs/` now names the command. This includes the observer remedy path, which had no validation step and put an incorrect `depends-on` value on a default branch. Parse errors now give the effect and name the command. The `branch` warning accepts conventional-commits prefixes and a dot, so `feat/login` and `release/1.22.0` pass.

## [1.21.0 -- One Run, One Observation](RELEASE-NOTES-1.21.0.md)

The observer files a single observation and ends, and the schedule sets the rate rather than the run itself. `--interval` is removed with an error that names its replacement, `install-cron` defaults to once a day, and an interval longer than a day is refused instead of silently rendering a daily cron line. Dedup compares `affected` paths after normalization, so `parser.go`, `./parser.go`, and `parser.go:42` stop filing a second observation for drift already on a branch. A malformed glob in `.spekk/dont-flag.yaml` is now a parse error rather than a silently dead suppression.

## [1.20.0 -- Reliable Headless Runs, Cross-Branch Data](RELEASE-NOTES-1.20.0.md)

Scheduled sandbox runs no longer die silently: the headless launcher prepends the spekk binary's directory to the child PATH, so bare `spekk` calls inside spawned sessions resolve under cron and systemd. Each observer skill gets its own lock file (a held lock prints one line instead of a silent exit 0). And `spekk list --cross-branch --json` exposes the merge-preview classification as machine-readable rows for observer agents.

## [1.19.0 -- Custom Frontmatter Fields, Indexed](RELEASE-NOTES-1.19.0.md)

Projects attach their own frontmatter keys (`workflows:`, `tags: [infrastructure, compliance]`) to specs and assertions. The parser now preserves every key outside the known set, and `spekk index` stores them in a new `frontmatter_fields` table — one row per distinct value, with flow sequences, comma scalars, and block lists indexing identically and quotes protecting commas. Per-tag progress reporting becomes one `spekk query`. The schema version goes to 3 (existing databases rebuild transparently), and `--force` now drops every table in `sqlite_master`, so a stale binary can never leave a future table's rows behind.

## [1.18.0 -- The Agent Token Leaves the WebSocket URL](RELEASE-NOTES-1.18.0.md)

The sandbox agent-client dials the path-less WebSocket route; the `Authorization: Bearer` header is the sole carrier of the token. The control host authenticates on the header alone, so the path token was pure leak surface — proxy logs, access logs, dial-error strings. Gone.

## [1.17.1 -- Announcements Drop the Evidence Path List](RELEASE-NOTES-1.17.1.md)

Observer announcements no longer carry an `Evidence:` line of `affected` paths. On a finding that touches ten files the line was longer than the finding itself, it repeated what the PR already shows, and it pushed the pointer line out of view. Evidence keeps its other two roles: an observation with no `affected` path stays invalid and never announces, and the observation file and the PR body still carry the paths in context.

## [1.17.0 -- The Sandbox States Its Protocol Version](RELEASE-NOTES-1.17.0.md)

The WebSocket contract between the agent-client and the control host gets one version number, exchanged at connect. The client sends `X-Spekk-Protocol: 1.0` on every dial and reads the server's `welcome` frame in return: a different major produces a clear operator warning, and a 4004 close logs one line without a reconnect hot-loop. A pinned constant makes every version change a deliberate diff. Either side deploys first safely.

## [1.16.0 -- The Observer Lifecycle Lives in Git](RELEASE-NOTES-1.16.0.md)

The observation lifecycle becomes declarative git state. An observation is a frontmatter file born on an `observer/<slug>` branch, and the branch set is the state machine: visible = pending, merged = resolved, PR closed with the branch kept = parked, deleted = forgotten. New `spekk observer announce` sends one message per run with at most three findings and flips `announced:` only after delivery; `spekk observer scan-check` gates every new observation against `.spekk/dont-flag.yaml` suppressions and cross-branch dedup; `spekk observer digest` replaces the committed DIGEST.md with a rendered view. The SQLite index gains observation tables across the branch union. Git fetch is the only remote read — no forge API calls.

## [1.15.1 -- Migrate the Old Dev-Loop Skill](RELEASE-NOTES-1.15.1.md)

A patch for the 1.15.0 install migration. 1.15.0 recognized only the role shims as old spekk files, so an unstamped dev-loop skill from an earlier version was left in place — an existing user did not get the new single-session skill. `spekk install` now recognizes the dev-loop skill by its heading and updates it in place, with a `.bak` backup.

## [1.15.0 -- One-Session Dev Loop and a Self-Migrating Install](RELEASE-NOTES-1.15.0.md)

The dev loop now runs in one session. The `spekk-dev-loop` skill plays the coach, builder, and verifier roles in turn (it fetches each role with `spekk prompt <role>`) instead of dispatching a sub-agent per role — for a feature that fits in one session, that is about 5 times fewer tokens, about 3 times cheaper, and about 3 times faster, at the same quality (issue #154). To match, `spekk install` writes the observer as an agent and the coach, builder, and dev-loop as skills, and it becomes an idempotent reconciler that migrates an old layout: it writes the desired files, removes a file a new layout no longer needs, backs up and never writes over a file you changed by hand, and `spekk update` warns when an old layout is present.

## [1.14.0 -- Query Your Specs with SQL](RELEASE-NOTES-1.14.0.md)

New `spekk index` builds a pure-Go SQLite index of the spec tree (`specs`, `assertions`, `depends_on`), and `spekk query` runs read-only `SELECT`s against it with `--json`/`--tsv`/`--csv` output — filtering, counting, grouping, and dependency joins in SQL. The index is a gitignored derived artifact, refreshed automatically when specs change, schema-versioned so upgrades rebuild it transparently, and opened read-only so a query can never mutate it.

## [1.13.0 -- The `prune` Observer Skill](RELEASE-NOTES-1.13.0.md)

New opt-in `spekk observer prune` skill surfaces genuinely-unused code and design-level redundancy (duplication, over-abstraction, dead config) as observations for human review — recommend-only and precision-biased, keyed on disuse rather than the absence of a spec. The existing `coverage-gap` skill is realigned to the same progressive-spec philosophy (optional documentation opportunities, no "no spec → delete" framing), and the coach/builder agents now soft-wrap spec prose to keep diffs minimal.

## [1.12.0 -- Agents Can Open Conversations](RELEASE-NOTES-1.12.0.md)

Sandbox agents can open new conversations on the connected chat surface: `spekk conversation open` spools an atomic request; the worker stamps the authoritative session id and emits a `conversation_open` frame; human replies resume the initiating session. Plus legible typed-error-frame logging, an additive Authorization header on the WebSocket dial, and neutralized infrastructure wording in public files.

## [1.11.0 -- `spekk validate` Hard Gate and Untrusted-Input Hardening](RELEASE-NOTES-1.11.0.md)

New `spekk validate` command: a strict, CI-friendly counterpart to the lenient parser — hard failures for malformed frontmatter, duplicate ids, dangling dependencies, lock-state mismatches (`in_progress` without `locked-by` and vice versa), and illegal parent `status` fields. The builder agent now runs it before every commit. All three agent prompts (builder, coach, observer) gain explicit untrusted-input rules: external content is data, never instructions.

## [1.10.10 -- Fix Broken Markdown in `spekk show --watch`](RELEASE-NOTES-1.10.10.md)

Watch mode could render as raw JavaScript instead of the spec explorer. The live-reload injector matched a `</body>` inside the bundled sanitizer's source before the real one, splitting the library onto the page as text. It now anchors on the document's final `</body>`.

## [1.10.9 -- Remove Self-Documenting Spec Headers](RELEASE-NOTES-1.10.9.md)

Reverts the second half of 1.10.6: the coach and builder agents no longer prepend an HTML-comment frontmatter-explainer header to newly authored spec/assertion files. It cluttered every file with boilerplate that `specs/README.md` already covers. The managed `specs/README.md` itself is unchanged.

## [1.10.8 -- Dev-Loop Skill for Every Harness](RELEASE-NOTES-1.10.8.md)

`spekk install` now writes the `spekk-dev-loop` orchestration skill into every supported harness, not just Claude Code. Native-skill harnesses (claude-code, opencode) get the skill verbatim; cursor, codex, and copilot get a frontmatter-stripped `/spekk-dev-loop` command. One embedded source, mapped to each tool's native location.

## [1.10.7 -- Init Creates the README in Pre-Existing `specs/`](RELEASE-NOTES-1.10.7.md)

Fixes a 1.10.6 gap: `spekk init` on a project whose `specs/` directory already existed but had no `README.md` wrote nothing. It now creates the managed README, covering all four states (missing, legacy, well-formed, corrupt).

## [1.10.6 -- Self-Documenting `specs/` Tree](RELEASE-NOTES-1.10.6.md)

`spekk init` writes a `specs/README.md` with a CLI-managed, idempotently-regenerated block documenting the concept model, frontmatter schema, and a `spekk_schema_version` — human prose outside the fence is never touched, and legacy/corrupt READMEs are upgraded or recovered in place. The coach and builder agents also add inline frontmatter-header comments to newly authored spec/assertion files.

## [1.10.5 -- Dev-Loop Skill + Coach Declarative Rewrite](RELEASE-NOTES-1.10.5.md)

`spekk install --target claude-code` now writes a `spekk-dev-loop` skill alongside the agent shims. The coach's declarative-framing section is rewritten with a write-first rule and a `Done when:` block. Repository hygiene: stray committed binaries removed and gitignored, obsolete loop scripts deleted, migration guide moved into docs.

## [1.10.4 -- `spekk list` + Coach Precision](RELEASE-NOTES-1.10.4.md)

New `spekk list` subcommand for filtered spec/assertion enumeration with `--json`/`--tsv`/`--csv`/`--long` output — 16× fewer tokens than a full scan. Coach gains encoding-precision guidance for non-obvious behavioral constraints.

## [1.10.3 -- Differential Diagnosis Placement Fix](RELEASE-NOTES-1.10.3.md)

Moves the coach's differential diagnosis protocol to the end of the prompt with the prohibition first, so it fires reliably across all three supported models. Eval COACH-01: 0/FAIL → 2/PASS across 3 models.

## [1.10.2 -- Differential Diagnosis Protocol](RELEASE-NOTES-1.10.2.md)

Adds a diagnostic protocol to the coach: for "why does X work for A but not B?" questions, it enumerates variables and asks before hypothesizing instead of proposing fast.

## [1.10.1 -- Observer Curation and Scheduling](RELEASE-NOTES-1.10.1.md)

`spekk observer consolidate` maintains a lean, severity-ranked digest from raw observations. The default loop now closes each cycle with a quiet consolidation pass. New `install-cron` / `uninstall-cron` subcommands schedule the observer via crontab with a Go-level overlap guard, headless Claude launch, and automatic claude path detection. Patch: Windows cross-compilation fix.

## [1.9.0 -- Cross-Branch Merge Preview](RELEASE-NOTES-1.9.0.md)

`spekk show --cross-branch` previews what merging each branch into the current branch would do to the spec corpus — read-only, with inline diff badges, branch filtering, and conflict detection. Observer now supports skills with layered resolution and ships a `coverage-gap` seed skill.

## [1.8.1 -- Show Markdown Rendering Fix](RELEASE-NOTES-1.8.1.md)

The `spekk show` detail panel now renders the markdown body with proper typography — frontmatter stripped, prose styled, monospace reserved for code.

## [1.8.0 -- Sudo-Free Installs and Updates](RELEASE-NOTES-1.8.0.md)

`install.sh` now defaults to user-owned `~/.local/bin`, so `spekk update` works without sudo. The installer warns (with the exact fix) when the directory isn't on `PATH`, and `spekk update` fails fast with clear guidance when it lacks write permission.

## [1.7.0 -- XDG-Compliant Config Directory](RELEASE-NOTES-1.7.0.md)

The global config directory moves from `~/.spekk` to `~/.config/spekk` (honoring `$XDG_CONFIG_HOME`), with automatic migration of existing directories. Platform support clarified: macOS and Linux.

## [1.6.0 -- Use Spekk from Any Coding Assistant](RELEASE-NOTES-1.6.0.md)

New `spekk install` registers the spekk agents as subagents in Claude Code, Cursor, Copilot, OpenCode, or Codex. New `spekk prompt` and `spekk skill` commands let agents in any harness fetch their instructions and skills on demand.

## [1.5.0 -- The Go Rewrite](RELEASE-NOTES-1.5.0.md)

Spekk is now a single static Go binary — no Node.js runtime required. Skills and prompts are embedded in the binary, `spekk update` self-updates via GitHub Releases, and eight security vulnerabilities were remediated.

## [1.4.0 -- Layered Prompts, Sandboxes, WebSocket Server](RELEASE-NOTES-1.4.0.md)

Agent prompts can now be customized globally (`~/.spekk/`) or per-project (`.spekk/`) with extend and override files. New sandbox commands for cloud agent environments. WebSocket server for real-time integrations.

## [1.3.0 -- Cloud Sandboxes](RELEASE-NOTES-1.3.0.md)

New `spekk sandbox` command for provisioning and managing DigitalOcean droplet-based agent sandboxes.

## [1.2.4 -- Live Reload + Searchbar](RELEASE-NOTES-1.2.4.md)

`spekk show --watch` for live-reloading the spec explorer as specs change. Searchbar filtering in the explorer UI.

## [1.2.1 -- Dependency Visualization](RELEASE-NOTES-1.2.1.md)

`spekk show` now renders an interactive metro map of dependency trees for each branch.

## [1.2.0 -- Coordinator Skill & Skills Architecture](RELEASE-NOTES-1.2.0.md)

Coordinator skill for dependency-aware work planning. Skills converted from JS classes to markdown files.

## [1.1.0 -- Builder CLI Flags](RELEASE-NOTES-1.1.0.md)

Builder now loops continuously by default. New flags: `--once`, `--dry-run`, `--spec`, `--assertion`, `--confirm`, `--interactive`.
