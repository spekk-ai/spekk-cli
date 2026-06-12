# Spekk CLI 1.8.0 — Sudo-Free Installs and Updates

This release covers changes since v1.7.0.

## Install Defaults to ~/.local/bin (PR #114)

`install.sh` now installs to `~/.local/bin` (user-owned) instead of `/usr/local/bin`. The payoff: `spekk update` works without sudo, every time.

This follows the convention of modern self-updating CLIs (rustup, deno, bun, uv) — a tool that replaces its own binary belongs in a directory the user owns. In-binary sudo escalation was considered and rejected.

Details:

- The install directory is created automatically if missing
- If `~/.local/bin` isn't on your `PATH`, the installer prints the exact `export PATH=...` line and which rc file to add it to (it never edits your dotfiles)
- `SPEKK_INSTALL_DIR` still overrides the location — `SPEKK_INSTALL_DIR=/usr/local/bin` restores the old behavior, with the existing sudo fallback
- If another `spekk` binary shadows the new install (e.g. a previous sudo install in `/usr/local/bin`), the installer points it out

**Existing installs are not moved.** If you previously installed to `/usr/local/bin`, either keep updating with `sudo spekk update`, or re-run the installer to switch to `~/.local/bin` (then remove the old binary: `sudo rm /usr/local/bin/spekk`).

## Actionable Error When `spekk update` Lacks Permission (PR #114)

For installs that remain in root-owned directories, `spekk update` previously failed with a raw permission error — after already initiating the download. Now it fails fast, before any network request, with clear guidance:

```
no write permission for /usr/local/bin (spekk was likely installed with sudo) — run: sudo spekk update, or reinstall to user-owned ~/.local/bin for sudo-free updates: https://github.com/spekk-ai/spekk-cli#install
```

**Note for users on 1.5.0–1.7.0:** the version of `spekk update` you're running predates this fix, so you'll see the old raw error. The fix for your update is the same: `sudo spekk update`.

## New Specs

The self-update command and install script are now covered by specs (`specs/self-update/`, `specs/install-script/`), including the rationale for the user-owned-directory convention.

## Upgrade

```bash
sudo spekk update   # if installed to /usr/local/bin
spekk update        # if installed to a user-writable directory
```

Or re-run the installer to switch to the new sudo-free location:

```bash
curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh
sudo rm /usr/local/bin/spekk   # remove the old binary if you had one
```
