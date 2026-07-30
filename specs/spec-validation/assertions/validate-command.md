---
id: validate-command
parent: spec-validation
created: 2026-07-23T21:10:51Z
priority: 1
status: done
---

# `spekk validate` enforces spec invariants with a non-zero exit gate

Running `spekk validate` checks every spec and assertion under `specs/` against a
fixed set of invariants and exits non-zero if any is violated. It reuses the
`internal/parser` model rather than reimplementing frontmatter parsing.

## Success criteria

### Command wiring
- `spekk validate` is a subcommand in `cmd/spekk/main.go`, dispatched from the
  top-level `switch` like the other commands, and listed in `spekk help`.
- The validation logic lives in a new `internal/validate/` package (mirroring
  `internal/status/`), which imports `internal/parser` and calls the exported
  `ParseSpecContent` / `ParseAssertionContent` (and/or `ParseAllSpecs`) — it does
  **not** contain its own frontmatter parser.
- It resolves the specs directory the same way other commands do (git-root
  `specs/`, `--specs-dir` override honored).

### Invariants checked (all of them, in one pass)
1. **Frontmatter well-formedness (strict).** Every `.md` file under `specs/*/`
   that begins with frontmatter must parse without error: valid kebab-case `id`,
   ISO-8601 UTC `created`, integer `priority` in 1–3, `status` in
   {not_started, in_progress, done, failed, draft}, and required fields present
   (assertions need `parent`). A file the lenient parser would skip-with-warning
   is a **failure** here. Files with no frontmatter are ignored (not every `.md`
   is a spec file).
2. **Parent resolution.** Every assertion's `parent` names an existing spec.
3. **Dependencies.** Every `depends-on` is kebab-case, references an existing
   assertion, is not self-referential, and participates in no cycle.
4. **Uniqueness.** No two specs share an `id`; no two assertions share an `id`
   (the parser only *warns* on duplicate assertion ids — validate fails).
5. **Lock-state pairings.** For every assertion:
   `status: in_progress` ⟹ non-empty `locked-by`; and
   `status: done|failed|not_started|draft` ⟹ **no** `locked-by`.
6. **Parent has no rolled-up status.** A parent spec file's frontmatter has no
   `status` field, OR its only value is `draft`. Any other value
   (`done`, `in_progress`, `failed`, `not_started`) is a failure, because the
   parser silently overwrites it. **Detection uses the raw frontmatter, not the
   parsed `Spec.Status`:** `ParseSpecContent` defaults an *absent* `status` to
   `"not_started"`, so it cannot distinguish "no status field" (pass) from a
   literal `status: not_started` (fail). Validate checks whether the `status:`
   key is literally present in the parent's frontmatter — a key-presence/value
   check, which is not the same as re-implementing the frontmatter parser.

### Exit-code and output contract
- **All invariants hold:** exit code `0` and one concise success line to stdout,
  e.g. `validate: N specs, M assertions OK`.
- **Any invariant fails:** exit code `1`. Every failure is reported (not just the
  first), one per line, each identifying the offending file path and the specific
  problem, e.g.
  `specs/foo/assertions/bar.md: status is in_progress but locked-by is missing`.
  Failures are printed deterministically (sorted by file path, then by message)
  so output is stable across runs and diffable in CI.
- Validate's own report is on **stdout** and is clean: an all-valid run prints
  only the summary line; a run with failures prints only the sorted failure
  lines (plus the non-zero exit). Validate itself introduces no spurious
  diagnostics into that report. Advisory warnings the *reused parser* already
  emits to stderr (e.g. `validateBranch`'s "non-standard branch pattern" notice,
  which fires for many existing assertions) are not validation failures, must
  not appear as failure lines, and validate is not required to suppress them.

### Tests
- `internal/validate/` has a test (`validate_test.go`) that drives validation
  over fixture spec trees and asserts BOTH the returned failures AND the exit
  code for: an all-valid tree (exit 0); an `in_progress` assertion missing
  `locked-by` (fail); a `done` assertion that still has `locked-by` (fail); a
  `priority: 4` / bad-status file (fail, and it is *not* silently skipped);
  a duplicate assertion id (fail); a parent spec with `status: done` (fail) vs a
  parent with `status: draft` (pass); and a dangling `depends-on` (fail).
- `spekk next` and `spekk list` behavior is unchanged — their leniency is not
  touched. Existing parser tests still pass (`go test ./...`).

**Note:** The failure report's value is the exit code (the CI/loop contract) plus
enough text for a human to fix the file. Keep it plain text — no JSON, no
severity levels, no config flags in v1.

**Tests:** `internal/validate/validate_test.go` — all-valid tree (pass);
`in_progress` missing `locked-by` (fail); `done` with `locked-by` still set
(fail); malformed assertion (bad priority/status) failing instead of being
silently skipped; duplicate assertion id (fail); parent `status: done` (fail)
vs parent `status: draft` (pass); the absent-status-vs-explicit-`not_started`
regression case; dangling `depends-on` (fail); missing `specs/` dir (pass,
trivially). `cmd/spekk/main_test.go` (new) covers the CLI wiring: exit codes
and stdout content for `execValidate` against a clean vs. broken fixture.
