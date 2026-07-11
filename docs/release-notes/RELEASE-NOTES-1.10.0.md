# Spekk CLI 1.10.0 — Observer Curation and Scheduling

This release covers changes since v1.9.0.

## Curator-Not-Firehose Observer (PR #5)

Early testing showed the observer generating far more observations than users could act on. This release introduces a two-stage curation model: raw observations stay cheap and private; a dedicated consolidation pass produces a single lean digest as the only user-facing surface.

### `spekk observer consolidate`

A new built-in observer skill that reviews all raw observations across every `observations/*/` subdirectory and rewrites a curated `observations/DIGEST.md`:

```bash
spekk observer consolidate
```

What it does:

- **Forced deliberation** — reads every open observation before any pruning decision; concluding "nothing to prune" without the full review is a contract violation
- **Archive, never delete** — stale or duplicate observations are moved to `observations/archive/` with originals preserved
- **Capped digest** — `observations/DIGEST.md` holds at most 5 open items, ranked high → medium → low, each linking to the underlying raw observation file
- **Mandatory rewrite** — the digest is rewritten on every run, even a quiet one, so it always reflects the latest state

### Digest as default surface

The observer's default monitoring loop now closes each scan cycle with a consolidation pass and reports only a brief summary from `DIGEST.md` (open item count and severities). When the digest is empty, the observer stays silent. Raw observation files are still written to `observations/default/` as before.

### `spekk observer install-cron` / `uninstall-cron`

New subcommands that install crontab entries so the observer runs on a schedule without manual intervention:

```bash
spekk observer install-cron                                   # Defaults: loop every 30 min, consolidate every 6 h
spekk observer install-cron --loop-interval 60 --consolidate-interval 720
spekk observer uninstall-cron                                 # Removes only spekk-managed entries
```

The installed entries:

- Change into the project directory before running (cwd captured at install time)
- Detect `claude` at install time and embed its absolute path — fails clearly if `claude` isn't on PATH rather than installing a broken entry
- Run Claude in headless mode (no TTY required)
- Use a Go-level overlap guard (`syscall.Flock`) on a project-scoped lock file under `.spekk/` — a new cron invocation skips silently if the previous session is still running
- Append output to `.spekk/observer.log` / `.spekk/observer-consolidate.log`

Interval validation rejects values that cron can't express exactly (e.g. 90 minutes); only values ≤ 60 or exact multiples of 60 are accepted.

`uninstall-cron` removes only the entries it added (identified by a `# spekk-observer` marker), leaving the rest of the crontab untouched.

## Upgrade

```bash
spekk update        # if installed to a user-writable directory
sudo spekk update   # if installed to /usr/local/bin
```
