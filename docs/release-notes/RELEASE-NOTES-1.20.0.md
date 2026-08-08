# Spekk CLI 1.20.0 — Reliable Headless Runs, Cross-Branch Data

Three changes, all born from running scheduled observers in production.

## The headless child session finds `spekk`

Scheduled sandbox runs died silently: the dispatcher invokes spekk by absolute path, but agent prompts run bare `spekk` commands inside the spawned headless session — and cron/systemd environments do not carry the install directory on PATH. The session failed with "command not found" seconds in, while the dispatcher saw a normal exit. `LaunchHeadless` now spawns Claude with the spekk binary's own directory prepended to the child PATH.

## Per-skill lock files for headless observer runs

All headless observer runs except `consolidate` shared one lock file, so a scheduled custom skill run exited 0 silently whenever the default loop was active. Each skill now locks its own `.spekk/observer-<skill>.lock` (the default loop keeps `observer-loop.lock`, `consolidate` keeps its path), and a held lock prints one line naming the lock file before the exit 0 — a cron log that shows nothing cannot be told apart from a run that never happened.

## Cross-branch spec state as data

`spekk list --cross-branch [--branch-filter <glob>]` emits the merge-preview classification — one row per changed (file, branch) pair with `path`, `branch`, `state`, `degraded`, and `old_status`/`new_status` drift — as a table, `--json`, `--tsv`, or `--csv`. The HTML explorer (`spekk show --cross-branch`) stays the human surface; this is the one an observer agent consumes. The engine's read-only guarantee applies unchanged.

## Upgrade

```bash
spekk update
```
