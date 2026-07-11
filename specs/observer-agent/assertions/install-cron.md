---
id: install-cron
parent: observer-agent
branch: observer-reimpl
created: 2026-07-11T15:00:00Z
priority: 2
status: done
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

### Installed cron lines are actually runnable

The generated crontab lines must work when cron executes them — cron runs in
`$HOME` with no TTY, so the naive `<schedule> spekk observer` line is
non-functional. Each installed line must satisfy all of:

- **Runs in the project directory.** The current working directory is
  captured at install time and each line changes into it before running the
  observer, e.g. `cd '<project-dir>' && <binary> observer ...`. The project
  directory is quoted so paths with spaces survive.
- **Runs headless.** The observer ultimately shells out to
  `claude --dangerously-skip-permissions`, which needs a TTY that cron does
  not provide. Cron-launched sessions must run Claude in print/headless mode
  (`claude -p`) rather than the interactive path, and redirect output to a
  log file under the project (e.g. `>> <project-dir>/... 2>&1`) instead of
  interactive stdout. (Whether the `-p` switch is applied by a cron-only flag
  the installed line passes to `spekk observer`, or by the observer detecting
  a non-interactive session, is the builder's call — the observable
  requirement is that a cron invocation never blocks on an absent TTY.) The
  output is redirected to a log under the project's `.spekk/` directory (e.g.
  `>> <project-dir>/.spekk/observer.log 2>&1`).
- **`claude` is resolved to an absolute path at install time.** The headless
  path launches `claude`, but cron's `PATH` is typically just `/usr/bin:/bin`
  while `claude` normally lives outside it (e.g. `~/.claude/local/claude`), so
  a bare `claude` name lookup fails silently under cron. `install-cron`
  resolves the absolute path via `exec.LookPath("claude")` at install time and
  bakes that absolute path into what the installed line runs. If
  `exec.LookPath("claude")` fails, `install-cron` exits with a clear error and
  installs nothing — it must never write a cron entry that is already known to
  be broken.
- **The `.spekk/` directory exists before the crontab is written.** The
  installed line redirects into `<project-dir>/.spekk/` and the Go lock files
  live there too, so `RunObserverInstallCron` creates
  `<project-dir>/.spekk/` (if absent) before writing the crontab entry;
  otherwise the shell redirect fails and the cron command crashes before
  spekk runs.
- **Guards against overlap (in Go, not shell `flock`).** The observer prompt
  defines an infinite monitoring loop, so launching a fresh session every
  interval would pile up unbounded concurrent sessions. The overlap guard is
  implemented in Go, not via a shell `flock` wrapper on the cron line —
  `flock(1)` is a util-linux CLI that macOS does not ship, so a `flock -n ...`
  cron line would fail with `flock: command not found` on every macOS run. The
  installed cron line therefore contains no `flock`. Instead, the headless
  launch path (`spekk observer --headless` / the internal `LaunchHeadless`)
  acquires a per-project lock via `syscall.Flock` with `LOCK_EX | LOCK_NB` on
  an `O_CREATE | O_RDWR` file descriptor before launching Claude.
  `syscall.Flock` is available on both macOS and Linux and needs no external
  binary; the kernel releases the lock automatically when the process exits. A
  new invocation that cannot acquire the lock exits 0 silently (this is the
  normal "already running" case, not an error). The lock files are
  project-scoped, living under `<project-dir>/.spekk/` (e.g.
  `.spekk/observer-loop.lock` and `.spekk/observer-consolidate.lock`), never
  under a global path like `/tmp` — otherwise two projects on the same machine
  would share one lock and silently starve each other. The loop entry and the
  consolidate entry use distinct lock files so they do not block each other.

### Interval validation rejects non-cron-expressible values

`ParseInstallCronFlags` rejects intervals that cron cannot express exactly,
returning a clear error instead of emitting a silently-wrong schedule
(today `minutesToCron(90)` yields `*/90 * * * *`, which strict crons reject
and lax crons misinterpret). An interval is accepted only if it is either
`<= 60` (rendered as a sub-hourly `*/N * * * *`) or an exact multiple of 60
(rendered as an hourly `0 */H * * *`). Values such as 90 or 45-when->60 are
rejected at parse time for both `--loop-interval` and `--consolidate-interval`.

### Cron line robustness

- **Binary path is quoted.** `buildCronLines` emits the spekk binary path
  wrapped in quotes so a path containing spaces (common under macOS home
  directories) does not break the cron command.
- **Crontab detection is locale-independent.** The `crontab -l` invocation in
  `readCrontab` runs with `LC_ALL=C` (or equivalent) so the "no crontab"
  empty-crontab sentinel is matched regardless of the user's locale.
