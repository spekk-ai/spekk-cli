---
id: install-script
created: 2026-06-11T00:00:00Z
priority: 2
---

# One-Command Install Script

A developer reading the README finds a single copy-paste command that installs spekk. No platform tables, no multi-step instructions at the top of the funnel.

```bash
curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh
```

## Install Location: User-Owned by Default

The default install directory is `~/.local/bin` (created if missing), not
`/usr/local/bin`.

**Rationale:** spekk is a self-updating CLI (`spekk update`, see
`specs/self-update/`). A binary installed with sudo into a root-owned
directory cannot update itself without sudo, which breaks the update flow for
the common case. Self-updating CLIs belong in user-owned directories — this is
the established precedent set by rustup (`~/.cargo/bin`), deno (`~/.deno/bin`),
bun (`~/.bun/bin`), and uv (`~/.local/bin`). The alternative — auto-escalating
to sudo from inside the binary during `spekk update` — was considered and
rejected: a CLI silently invoking sudo on the user's behalf is a security
anti-pattern, and the fail-fast error message
(`specs/self-update/assertions/fail-fast-permission-error.md`) already covers
legacy sudo installs.

Users who prefer a system-wide install keep that option:
`SPEKK_INSTALL_DIR=/usr/local/bin` still works, with the existing sudo
fallback when the target directory is not writable.

The trade-off of `~/.local/bin` is that it is not on `$PATH` on every system,
so the script must detect that and tell the user exactly what line to add.

## Assertions

- `install-sh-works` — core install behavior (done)
- `default-install-dir-user-owned` — default moves to `~/.local/bin` (not yet implemented)
- `path-warning-after-install` — PATH check with copy-paste fix (not yet implemented)
