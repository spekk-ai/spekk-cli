# Spekk CLI — Install

## Quick install (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh
```

The script detects your platform, downloads the latest release, and installs to `~/.local/bin` (override with `SPEKK_INSTALL_DIR`, e.g. `SPEKK_INSTALL_DIR=/usr/local/bin` for a system-wide install). If `~/.local/bin` is not on your `PATH`, the script prints the exact line to add to your shell config. Verify:

```bash
spekk version
```

## Windows (PowerShell)

```powershell
$arch = if ([System.Environment]::Is64BitOperatingSystem) { "amd64" } else { "arm64" }
$url = "https://github.com/spekk-ai/spekk-cli/releases/latest/download/spekk-windows-${arch}.exe"
Invoke-WebRequest -Uri $url -OutFile "$env:LOCALAPPDATA\Microsoft\WindowsApps\spekk.exe"
```

## Manual download

Download the binary for your platform from the [Releases page](https://github.com/spekk-ai/spekk-cli/releases/latest):

| Platform              | Binary name                  |
|-----------------------|------------------------------|
| macOS (Apple Silicon) | `spekk-darwin-arm64`         |
| macOS (Intel)         | `spekk-darwin-amd64`         |
| Linux (x86_64)        | `spekk-linux-amd64`          |
| Linux (ARM)           | `spekk-linux-arm64`          |
| Windows (x86_64)      | `spekk-windows-amd64.exe`    |
| Windows (ARM)         | `spekk-windows-arm64.exe`    |

## Updating

```bash
spekk update          # install latest
spekk update --check  # preview without installing
```

`spekk update` replaces the binary in place, so it needs write access to the install directory. The default install location (`~/.local/bin`) is user-owned, so updates just work. If spekk lives in a root-owned directory like `/usr/local/bin` (e.g. from an older install or `SPEKK_INSTALL_DIR=/usr/local/bin`), run `sudo spekk update` — this applies to every update there, not just the first. To update without sudo, reinstall to the user-owned default:

```bash
curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh
```

(Ensure `~/.local/bin` is on your `PATH` — the installer tells you the exact line to add if it isn't.)

---

## Claude-assisted setup

Paste this into Claude Code:

```
Install the spekk CLI for me. Download the right binary from https://github.com/spekk-ai/spekk-cli/releases/latest for my platform, put it in ~/.local/bin/ (create the directory if needed), make it executable, and make sure ~/.local/bin is on my PATH.
```
