# Spekk CLI — Install

## Quick install (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh
```

The script detects your platform, downloads the latest release, and installs to `/usr/local/bin` (override with `SPEKK_INSTALL_DIR`). Verify:

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

`spekk update` replaces the binary in place, so it needs write access to the install directory. If spekk lives in a root-owned directory like `/usr/local/bin`, run `sudo spekk update` — this applies to every update there, not just the first. To update without sudo, install to a user-writable directory:

```bash
curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | SPEKK_INSTALL_DIR=~/.local/bin sh
```

(Ensure `~/.local/bin` is on your `PATH`.)

---

## Claude-assisted setup

Paste this into Claude Code:

```
Install the spekk CLI for me. Download the right binary from https://github.com/spekk-ai/spekk-cli/releases/latest for my platform and put it in /usr/local/bin/.
```
