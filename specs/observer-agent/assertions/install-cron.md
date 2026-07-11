---
id: install-cron
parent: observer-agent
branch: observer-reimpl
created: 2026-07-11T15:00:00Z
priority: 2
status: not_started
depends-on: digest-as-default-surface
---

# Observer Runs on a Cron Schedule via install-cron

## Description

A `spekk observer install-cron` subcommand exists that writes crontab entries
so the observer loop and consolidation run on a schedule without user
intervention. This is stage 3 of the sequencing ladder in the parent spec and
is explicitly a cron-based stopgap: the future upgrade to Go-owned scheduling
(enabling agent self-directed task queues) is noted in
`specs/observer-agent/observer-agent.md` and is out of scope here. This stage
requires Go changes to add the subcommand.

## Success Criteria

- `spekk observer install-cron` installs two crontab entries: one running
  `spekk observer` (default loop) and one running
  `spekk observer consolidate`
- Intervals are configurable via `--loop-interval` and
  `--consolidate-interval` flags, with sensible defaults (default loop every
  30 minutes, consolidation every 6 hours)
- The command prints what it installed, including the exact crontab lines,
  so the user can verify the schedule
- A companion `spekk observer uninstall-cron` subcommand removes exactly the
  entries that `install-cron` added, leaving the rest of the user's crontab
  untouched
- Works on macOS and Linux; there is no Windows requirement
