# 1.21.0 — One Run, One Observation

The observer changes shape. A run now files a single observation and ends, and how often that happens is set by the schedule rather than by the run itself. Two flags change with it, and one of them is removed.

## A run files one observation

`spekk observer` searches recent change on the default branch, files the first real drift it finds, and stops. If it covers the areas it chose and finds nothing, it stops and says so. Those are the only two endings.

Before, a run kept going until something stopped it. That was the design when the observer was a monitoring loop, and it did not survive contact with a real repository: one production run scanned for hours across many subagents and filed nine observations at once. Every one of them would have been found by a later run.

One observation is the unit a person reviews — one branch, one pull request, one decision.

A run reports in one line, always, including when it files nothing. Silence would be indistinguishable from a run that never happened.

## The schedule sets the rate

These are two knobs, not one. A run files one observation whatever the interval, so the interval decides how many observations arrive, never how thorough a run is.

**`spekk observer --interval` is removed.** It set a scan cadence inside a session that no longer runs continuously. Passing it now exits with an error naming the replacement:

```
$ spekk observer --interval 60
Error: --interval is no longer a flag.
cadence is now set by the schedule that runs the observer, not by the run itself.
```

The error matters. Without it the flag would have parsed and its value read as a skill name, launching the wrong thing quietly.

**`spekk observer install-cron` defaults to once a day**, for the scan and for consolidation, and consolidation is scheduled after the scan rather than beside it. Thirty minutes was right for a session that had to be interrupted by the next one; alongside the new cap it would file up to 48 observations a day.

An interval longer than a day is now refused. `--loop-interval 2880` used to be accepted and rendered `0 */48 * * *`, which cron accepts and then runs daily at midnight — never every two days. An unknown argument, or the `--flag=value` form, is refused rather than ignored: it would otherwise install the default schedule and report success.

## Upgrading

**Existing crontabs are not migrated.** An entry written by an older `install-cron` keeps its 30-minute schedule, and it will not hit the removed-flag error because the installed line never carried `--interval`. After upgrading, that entry runs the new prompt on the old cadence.

Re-run `install-cron` to replace it:

```bash
spekk observer install-cron
```

It removes its own previous entries and writes new ones. `uninstall-cron` still finds the old entries.

## Dedup compares files, not spellings

Successive runs accumulate rather than repeat only if a run can tell that an earlier one already filed something. That comparison was exact string equality on `affected` paths, and those paths are written by an agent.

A run that filed `internal/parser/parser.go` did not cover a later run naming `internal/parser/parser.go:42` or `./internal/parser/parser.go`. Each of those filed a second observation for drift already on a branch — a new branch and a new pull request, on every run, for as long as the drift lasted.

Paths are now compared after normalization: a `:line` or `:line:column` suffix, a `./` prefix, duplicate slashes, a trailing slash, and any `.` or `..` segment. A directory is deliberately not reduced to the files beneath it — one directory-level finding would otherwise hide every file-level finding under it.

`.spekk/dont-flag.yaml` gets the same treatment on the path side, where it matters more: suppressed drift never becomes an observation, so nothing lands on a branch to cover it next time. A suppression pattern is never rewritten — it is a glob, not a path, and cleaning it as one would turn `docs/../**` into a rule that suppresses everything.

A malformed glob in `dont-flag.yaml` is now a parse error. It previously validated and then matched nothing, so a suppression someone wrote and reviewed was silently dead.

## Also

- `spekk observer` help and the CLI reference describe a scan rather than a monitoring loop.
- Advisory skill outputs (`coverage-gap`, `prune`) commit to `observer-advisory/*` branches. The `observer/*` namespace belongs to the observation lifecycle, and a branch there joins the dedup union.
