---
id: default-install-dir-user-owned
parent: install-script
created: 2026-06-12T00:00:00Z
priority: 1
status: not_started
branch: feat/user-owned-install-dir
---

# Default Install Directory Is User-Owned (`~/.local/bin`)

## Description

`install.sh` installs to `~/.local/bin` by default so that the installed
binary is user-owned and `spekk update` works without sudo. The previous
default was `/usr/local/bin`, which made every self-update require sudo on a
standard install.

## Success Criteria

- With no environment overrides, `install.sh` installs to
  `$HOME/.local/bin/spekk`
- `~/.local/bin` is created (`mkdir -p`) if it does not exist
- No sudo is invoked anywhere in the default flow
- `SPEKK_INSTALL_DIR` still overrides the destination; when the override
  points at a directory the user cannot write (e.g. `/usr/local/bin`), the
  existing sudo fallback is retained ("Installing to <dir> (requires sudo)"
  + `sudo mv`)
- The script's usage comment and the README/`docs/install.md` install docs
  reflect the new default and the `SPEKK_INSTALL_DIR` escape hatch
- Script remains POSIX sh (no bashisms), so `curl ... | sh` stays safe

## Notes

Rationale recorded in the parent spec: user-owned directories are the
rustup/deno/bun/uv precedent for self-updating CLIs; auto-sudo escalation
from inside the binary was rejected. Open question for the implementer:
whether to also print a hint for users with an existing sudo-installed
`/usr/local/bin/spekk` (the stale copy could shadow the new one depending on
PATH order) — a one-line "found another spekk at <path>" warning would be in
the spirit of this spec but is not required by it.
