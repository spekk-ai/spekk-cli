---
icon: lucide/megaphone
---

# Release Notes

What's new in each version of Spekk CLI.

---

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
