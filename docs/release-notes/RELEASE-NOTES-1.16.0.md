# Spekk CLI 1.16.0 — The Observer Lifecycle Lives in Git

Observer state used to live in prompts and loose files. A digest file served as both the current view and the announce memory, and that dual role broke announcements in production. This release moves the whole observation lifecycle into declarative git state, and moves the announce mechanics from prose into Go.

## Observations are files on branches

An observation is a Markdown file with YAML frontmatter: `slug`, `type`, `severity`, `status`, `created`, `announced`, `pr`, and `affected` (the evidence paths — an observation with no evidence is invalid). It starts life on an `observer/<slug>` branch, with the proposed remedy as a separate commit.

The branch set is the state machine, and `git fetch` is the only remote read:

| State | Representation |
|---|---|
| Pending | `observer/<slug>` branch visible on origin |
| Resolved | branch merged; the remedy lands with `status: resolved` |
| Parked | PR closed, branch kept — never re-flagged |
| Forgotten | branch deleted — a persisting drift is found again |
| Suppressed | entry in `.spekk/dont-flag.yaml` on main |

No forge API calls anywhere. PR status is deliberately invisible.

## `spekk observer announce`

A deterministic announce step for cron use. It selects the top unannounced open observations from the index — high or medium only, high first, oldest first — and sends ONE conversation message that carries at most THREE findings. After delivery, it commits the `announced:` frontmatter flip to each branch and pushes. A failure appends to `.spekk/observer-conversation.log` and exits non-zero without the flip, so the next run retries. The command must run inside a sandbox session (it uses the conversation spool).

## `spekk observer scan-check`

The gate an observer runs before it creates any observation. It answers in JSON: `suppressed` (an active `.spekk/dont-flag.yaml` entry matches), `covered` (an observation on a visible branch already covers the drift), or `new` (proceed, with the slug to use). A malformed suppression file fails the check loudly — a broken safety file is never treated as empty.

## `.spekk/dont-flag.yaml`

Human-gated suppressions, committed on main. Each entry has a required `match` (path glob or slug pattern), a required `reason` and `by`, and an optional `until` date (end of day, UTC). The observer only reads the file. The sanctioned path to a permanent dismissal is a small reviewed PR that adds an entry.

## `spekk observer digest`

`observations/DIGEST.md` is gone. The digest is now a rendered view: `spekk observer digest` prints the open observations across the visible branch union, ranked by severity, five at most. `--json` gives the machine-readable form.

## Index additions

The SQLite index gains `observations` and `observation_files` tables, built across main and all fetched `observer/*` branches. The invariant holds: every table is rebuildable from plaintext, or safe to lose. Queries stay read-only.
