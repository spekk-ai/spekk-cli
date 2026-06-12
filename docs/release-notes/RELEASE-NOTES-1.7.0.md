# Spekk CLI 1.7.0 — XDG-Compliant Config Directory

This release covers changes since v1.6.0.

## Global Config Moves to ~/.config/spekk (PR #112)

The global configuration directory has moved from `~/.spekk` to follow the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/latest/):

| Path | Used when |
|------|-----------|
| `$XDG_CONFIG_HOME/spekk` | `XDG_CONFIG_HOME` is set |
| `~/.config/spekk` | otherwise (the default) |

This matches the convention used by `git`, `gh`, `npm`, and most modern CLI tools, and keeps your home directory uncluttered. The project-local `.spekk/` directory is unchanged.

Everything that lived in `~/.spekk` moves with it: global prompt overrides and extensions, global skills, sandbox metadata, SSH keys, and known_hosts files.

### Automatic migration

On the first command that touches global config, an existing `~/.spekk` directory is migrated automatically:

- **In a terminal**, spekk shows what's moving and asks you to press Enter before continuing.
- **In non-interactive contexts** (agent shims, pipes, `spekk serve`), the migration happens silently with a notice on stderr — nothing blocks.
- The migration is safe under concurrency: multiple spekk processes starting at once (e.g. agent shims launching together at session start) resolve cleanly.
- Saved sandbox SSH key paths that point into the old location are remapped automatically, so existing sandboxes keep working.

No action is required. If you've scripted against `~/.spekk` directly, update those paths to `~/.config/spekk`.

## Platform Support Clarified

The README now states explicitly that spekk supports **macOS and Linux**. Windows binaries are published on a best-effort basis but are untested and unsupported — [WSL](https://learn.microsoft.com/en-us/windows/wsl/) is the recommended way to run spekk on Windows.

## Other Changes

- Release notes published for 1.5.0 (the Go rewrite) and 1.6.0 (`spekk install`)
- New `internal/fsutil` package for shared filesystem helpers
- Repo-wide `gofmt` cleanup

## Upgrade

```bash
spekk update
```

Or download the latest binary from [GitHub Releases](https://github.com/spekk-ai/spekk-cli/releases).
