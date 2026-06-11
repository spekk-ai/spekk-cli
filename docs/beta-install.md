# Spekk CLI — Beta Install

You'll receive a **token** from us — a GitHub PAT scoped to download releases.

## 1. Install via curl

Export your token first:

```bash
export GH_SPEKK_TOKEN="<token-we-gave-you>"
```

Then pick your binary name and run:

### macOS / Linux

| Platform              | Binary name              |
|-----------------------|--------------------------|
| macOS (Apple Silicon) | `spekk-darwin-arm64`     |
| macOS (Intel)         | `spekk-darwin-amd64`     |
| Linux (x86_64)        | `spekk-linux-amd64`      |
| Linux (ARM)           | `spekk-linux-arm64`      |
| Windows (x86_64)      | `spekk-windows-amd64.exe`|
| Windows (ARM)         | `spekk-windows-arm64.exe`|

```bash
BINARY="spekk-darwin-arm64"  # ← change this for your platform

# Fetch the asset download URL from the latest release
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
spekk --version
```

## 2. Set up auto-update

Add the token we gave you to your shell profile:

**macOS / Linux (zsh):**
```bash
echo 'export GH_SPEKK_TOKEN="<token-we-gave-you>"' >> ~/.zshrc
source ~/.zshrc
```

**Windows (PowerShell):**
```powershell
[System.Environment]::SetEnvironmentVariable("GH_SPEKK_TOKEN", "<token-we-gave-you>", "User")
```

Then you can update anytime:

```bash
spekk update          # install latest
spekk update --check  # preview without installing
```

---

## Claude-assisted setup

Paste this into Claude Code:

```
Install the spekk CLI for me. Use the GitHub releases API at spekk-ai/spekk-cli to download the right binary for my platform. Add GH_SPEKK_TOKEN to my shell profile. Ask me for the token value.
```
