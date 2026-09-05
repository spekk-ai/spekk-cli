# Spekk CLI Install

## Quick install (macOS and Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh
```

The script detects your platform, downloads the latest release, and installs it to `~/.local/bin`. When that directory is not on your `PATH`, the script prints the line to add to your shell configuration. Verify the install:

```bash
spekk version
```

The script reads two environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SPEKK_INSTALL_DIR` | `~/.local/bin` | Where to put the binary. For example `SPEKK_INSTALL_DIR=/usr/local/bin` for a system-wide install, which needs sudo |
| `SPEKK_VERSION` | `latest` | The release tag to install, for example `v1.28.0`. Pin it in CI |

The script never uses sudo for a directory under your home. A root-owned file there would break `spekk update` later.

## Windows (PowerShell)

Windows binaries are published but not tested. Use [WSL](https://learn.microsoft.com/en-us/windows/wsl/) for the supported path. To install the native binary:

```powershell
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
$url = "https://github.com/spekk-ai/spekk-cli/releases/latest/download/spekk-windows-${arch}.exe"
Invoke-WebRequest -Uri $url -OutFile "$env:LOCALAPPDATA\Microsoft\WindowsApps\spekk.exe"
```

## Manual download

Download the binary for your platform from the [releases page](https://github.com/spekk-ai/spekk-cli/releases/latest):

| Platform              | Binary name                  |
|-----------------------|------------------------------|
| macOS (Apple Silicon) | `spekk-darwin-arm64`         |
| macOS (Intel)         | `spekk-darwin-amd64`         |
| Linux (x86_64)        | `spekk-linux-amd64`          |
| Linux (ARM)           | `spekk-linux-arm64`          |
| Windows (x86_64)      | `spekk-windows-amd64.exe`    |
| Windows (ARM)         | `spekk-windows-arm64.exe`    |

Make the file executable, name it `spekk`, and put it on your `PATH`.

## From source

```bash
go install github.com/spekk-ai/spekk-cli/cmd/spekk@latest
```

This needs Go 1.25 or later. A build from source reports `spekk version` as `dev`, and `spekk update` refuses to update it. Run `go install` again instead.

## Updating

```bash
spekk update          # Install the latest release
spekk update --check  # See what is available, install nothing
```

`spekk update` replaces the binary in place, so it needs write access to the install directory. The default location, `~/.local/bin`, is yours, so the update works. When spekk is in a root-owned directory such as `/usr/local/bin`, from an older install or from `SPEKK_INSTALL_DIR=/usr/local/bin`, run `sudo spekk update`, every time. To update without sudo, install again to the default location:

```bash
curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh
```

Make sure `~/.local/bin` is on your `PATH`, and remove the old binary, or the shell keeps finding it first. The installer warns you when another `spekk` is earlier on the `PATH`.

After an update, `spekk update` checks the files that `spekk install --target` wrote and tells you when one no longer matches the new binary.

## Claude-assisted setup

Paste this into Claude Code:

```
Install the spekk CLI for me. Download the right binary from https://github.com/spekk-ai/spekk-cli/releases/latest for my platform, put it in ~/.local/bin/ (create the directory if needed), make it executable, and make sure ~/.local/bin is on my PATH.
```
