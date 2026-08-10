---
id: install-sh-works
parent: install-script
created: 2026-06-11T00:00:00Z
priority: 1
status: done
---

# install.sh Installs the Latest Release on macOS and Linux

`install.sh` at the repo root is a POSIX-sh script that installs the latest spekk release with one command.

## Success Criteria

- Detects OS (`darwin`/`linux`) and architecture (`amd64`/`arm64` including `x86_64`/`aarch64` aliases); errors clearly on anything else, pointing Windows users to the releases page
- Downloads `spekk-<os>-<arch>` from `https://github.com/spekk-ai/spekk-cli/releases/latest/download/`
- Installs `spekk` to the default install directory, overridable via `SPEKK_INSTALL_DIR`; uses `sudo` only when the target directory is not writable (which directory is the default is specified by `default-install-dir-user-owned`)
- Cleans up its temp file on failure
- Prints the installed version and a pointer to getting started
- Works under plain `sh` (no bashisms) so `curl ... | sh` is safe
- README leads with this one-liner in the Install section; `docs/install.md` quick-install uses it too
- Released binaries report their real version (publish.yml builds `cmd/spekk` with `-ldflags "-X main.version=<tag>"`), so the script's "Installed spekk \<version\>" line is meaningful and `spekk update` works on installed binaries

**Tests:** verified by running `SPEKK_INSTALL_DIR=<tmpdir> sh install.sh` against the live latest release.
