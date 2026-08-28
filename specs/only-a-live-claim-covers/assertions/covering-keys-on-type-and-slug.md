---
id: covering-keys-on-type-and-slug
parent: only-a-live-claim-covers
created: 2026-08-28T12:00:00Z
priority: 1
branch: fix/only-a-live-claim-covers
depends-on: covering-needs-the-owning-branch
status: done
---

# The Dedup Key Is the Type and the Slug, Not a Shared File

Two findings in one file are two findings. The slug says which finding this is; the `affected` list says where it lives.

## Success criteria

- `Covers` takes the candidate type and the candidate slug, and reports true only when both equal the observation's own. It no longer reads `Affected`, and it no longer calls `NormalizePath`.
- `FindCovering` takes the same two values and returns the first live claim that matches.
- `spekk observer scan-check` passes `--slug` to the covering test. `--type` and `--affected` stay required flags: the type is half the key, and `--affected` is what `.spekk/dont-flag.yaml` matches on.
- `NormalizePath` and its behavior are unchanged. `internal/dontflag` is its remaining caller, and suppression needs it more than dedup did — suppressed drift never becomes an observation, so no branch exists to cover it next time.
- The observer prompt (`specs/observer-agent/observer.prompt.md`) no longer teaches path-overlap dedup. Two passages state the old rule and must state the new one: the guidance to treat a named file as closed for its type, and the reason each `affected` entry names one file.

**Note:** the type stays in the key because it is part of what the finding is. One file can hold `code_spec_misalignment` drift and `outdated_specs` drift at the same time, and neither covers the other.

**Tests:** `internal/observation/` — a live claim covers a candidate with its slug and type, and does not cover a candidate that shares a path but carries a different slug; it does not cover the same slug at another type. `cmd/spekk/` — `scan-check` for a new finding that names a file an open observation already names returns `clear`, and the same slug as that observation returns `covered`. Delete the covers tests that only exercised path overlap; the path spellings they pin now belong to the `NormalizePath` tests, which stay.
