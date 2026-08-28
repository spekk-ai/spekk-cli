---
id: validate-covers-every-skip
parent: parser-warnings-belong-to-the-caller
created: 2026-08-22T15:00:00Z
priority: 1
branch: fix/parser-warning-volume
status: done
---

# `spekk validate` Reports Every Skip the Parser Records

The summary line tells the reader to run `spekk validate` for the detail. That instruction must be true for every warning the parser can raise, and today it is false for three of them.

## Success criteria

Of the seven warnings that survive `parser-collects-warnings`, four are already reported by `validate` as failures: a malformed spec file, a malformed assertion file, an unreadable assertion file, and a duplicate assertion id. The remaining three are invisible to it, and each is added as a **failure**, because each one loses work:

- **A spec directory that holds assertion files but no main spec file.** The parser skips the whole directory, so every assertion in it is dropped from the queue. `validate` reports it against the missing file path:
  `specs/foo/foo.md: spec directory has assertion files but no main spec file`
  Today this surfaces only as a downstream `parent "foo" not found`, and only when an assertion inside names that parent.
- **A path named `assertions` that is not a directory.** `Run` currently calls `os.ReadDir` and `continue`s on the error, so the spec's assertions are silently absent:
  `specs/foo/assertions: expected a directory`
- **An `assertions/` directory that cannot be read.** Same silent `continue`, usually a permissions fault:
  `specs/foo/assertions: cannot read directory: <error>`

### What stays out
- A spec directory with **no** `assertions/` directory is not reported. It parses correctly and is a normal spec with no assertions yet. `parser-collects-warnings` deletes the false warning that called it a skip, and `validate` must not reintroduce the same mistake as a failure.
- A directory under `specs/` that holds no file with frontmatter is not a spec directory at all, and stays ignored.

### Contract
- These are failures, so each sets exit code 1 and prints on stdout with the existing sorted, one-per-line format. A skipped file is lost work, not an advisory.
- The existing warning channel on `Result` is untouched. The stranded-branch and stale-lock reports stay warnings on stderr.

**Tests:** `internal/validate/` — a spec directory with an assertion file and no main spec file fails with the message above; a `specs/foo/assertions` regular file fails; a spec directory with no `assertions/` directory **passes**; an unreadable `assertions/` directory fails; a clean tree still passes with exit 0.
