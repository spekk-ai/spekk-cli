---
id: path-warning-after-install
parent: install-script
created: 2026-06-12T00:00:00Z
priority: 2
status: done
depends-on: default-install-dir-user-owned
---

# Install Warns With a Copy-Paste Fix When the Install Dir Is Not on PATH

## Description

`~/.local/bin` is not on `$PATH` on every system. After installing,
`install.sh` checks whether the install directory is on `$PATH` and, if not,
prints a clear warning containing the exact line the user should add to
their shell config — no detective work required.

## Success Criteria

- After a successful install, the script checks whether the install
  directory (default or `SPEKK_INSTALL_DIR`) appears as a component of
  `$PATH` (exact component match, not substring)
- If it is on `$PATH`: no warning, output unchanged from today
- If it is not on `$PATH`, the script prints a warning that includes:
  - the directory that is missing from `$PATH`
  - the exact line to add, e.g.
    `export PATH="$HOME/.local/bin:$PATH"`
  - the rc file to add it to, chosen from the user's shell where reasonable:
    `$SHELL` ending in `zsh` → `~/.zshrc`, ending in `bash` → `~/.bashrc`
    (`~/.bash_profile` on macOS), otherwise a generic "your shell config"
  - a reminder to restart the shell or `source` the rc file
- The warning goes to stderr or stands out clearly from the success output;
  the script still exits 0 (the install itself succeeded)
- The post-install version line (`"$INSTALL_DIR/spekk" version`) keeps using
  the full path, so it works even before PATH is fixed
- PATH detection works under plain `sh` (no bashisms)

## Notes

The script only prints the instruction — it must not edit the user's rc
files. Modifying dotfiles from a piped `curl | sh` script was deliberately
left out of scope.
