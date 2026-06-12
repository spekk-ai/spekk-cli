# Spekk CLI 1.7.1 — Update Permission Fix

This release covers changes since v1.7.0.

## Actionable Error When `spekk update` Lacks Permission (PR #114)

`spekk update` replaces the binary in place, which requires write access to the install directory. On a default install to `/usr/local/bin` (root-owned), running it as a regular user failed with an unhelpful raw permission error — after already initiating the download.

Now it fails fast, before any network request, with clear guidance:

```
no write permission for /usr/local/bin (spekk was likely installed with sudo) — try: sudo spekk update
```

**Note for users on 1.5.0–1.7.0:** the version of `spekk update` you're running predates this fix, so you'll see the old raw error. The fix for your update is the same: `sudo spekk update`. Installs in a root-owned directory will need sudo for every update; to update without sudo, install to a user-writable directory instead (`SPEKK_INSTALL_DIR=~/.local/bin`).

## Upgrade

```bash
sudo spekk update   # if installed to /usr/local/bin
spekk update        # if installed to a user-writable directory
```

Or download the latest binary from [GitHub Releases](https://github.com/spekk-ai/spekk-cli/releases).
