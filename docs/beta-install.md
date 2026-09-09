# Spekk CLI Beta Install

**This page is for the private beta, which ended when the repository became public.** You no longer need a token to download a release. Use the [install guide](install.md) instead. The steps below still work with a GitHub token, and stay here for anyone who follows an old link.

You receive a **token** from us: a GitHub personal access token that can download releases.

## 1. Install with curl

Export the token first:

```bash
export GH_SPEKK_TOKEN="<token-we-gave-you>"
```

Then pick your binary name and run the commands below.

### macOS and Linux

| Platform             | Binary name             |
|----------------------|-------------------------|
| macOS (Apple Silicon)| `spekk-darwin-arm64`    |
| macOS (Intel)        | `spekk-darwin-amd64`    |
| Linux (x86_64)       | `spekk-linux-amd64`    |
| Linux (ARM)          | `spekk-linux-arm64`    |
| Windows (x86_64)     | `spekk-windows-amd64.exe` |
| Windows (ARM)        | `spekk-windows-arm64.exe` |

```bash
BINARY="spekk-darwin-arm64"  # Change this for your platform

# Get the asset download URL from the latest release
ASSET_URL=$(curl -sL -H "Authorization: token $GH_SPEKK_TOKEN" \
  https://api.github.com/repos/spekk-ai/spekk-cli/releases/latest \
  | python3 -c "import sys,json; assets=json.load(sys.stdin).get('assets',[]); print(next(a['url'] for a in assets if a['name']=='$BINARY'))")

# Download the binary
curl -sL -H "Authorization: token $GH_SPEKK_TOKEN" -H "Accept: application/octet-stream" \
  "$ASSET_URL" -o spekk && chmod +x spekk && sudo mv spekk /usr/local/bin/
```

### Windows (PowerShell)

```powershell
$env:GH_SPEKK_TOKEN = "<token-we-gave-you>"
$headers = @{ Authorization = "token $env:GH_SPEKK_TOKEN" }
$release = Invoke-RestMethod -Uri "https://api.github.com/repos/spekk-ai/spekk-cli/releases/latest" -Headers $headers
$asset = $release.assets | Where-Object { $_.name -eq "spekk-windows-amd64.exe" }
$dlHeaders = @{ Authorization = "token $env:GH_SPEKK_TOKEN"; Accept = "application/octet-stream" }
Invoke-WebRequest -Uri $asset.url -Headers $dlHeaders -OutFile "$env:LOCALAPPDATA\Microsoft\WindowsApps\spekk.exe"
```

Verify:

```bash
spekk version
```

## 2. Updates

`spekk update` downloads from the public releases page and reads no token. Run it at any time:

```bash
spekk update          # Install the latest release
spekk update --check  # See what is available, install nothing
```

A binary in `/usr/local/bin` is root-owned, so the update there needs `sudo spekk update`. To update without sudo, install again to `~/.local/bin` with the [install script](install.md).

## Claude-assisted setup

Paste this into Claude Code:

```
Install the spekk CLI for me. Download the right binary from https://github.com/spekk-ai/spekk-cli/releases/latest for my platform, put it in ~/.local/bin/ (create the directory if needed), make it executable, and make sure ~/.local/bin is on my PATH.
```
