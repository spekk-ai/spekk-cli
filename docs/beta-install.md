# Spekk CLI — Beta Install

You'll receive a binary and a token directly from us.

## 1. Install

### macOS / Linux

```bash
chmod +x /path/to/spekk-<platform>
sudo mv /path/to/spekk-<platform> /usr/local/bin/spekk
```

### Windows

```powershell
Move-Item .\spekk-windows-amd64.exe C:\Users\$env:USERNAME\AppData\Local\Microsoft\WindowsApps\spekk.exe
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

Paste this into Claude Code along with the binary file you received:

```
Install the spekk CLI binary I'm dropping here. Make it executable, move it to the right location for my OS, and add GH_SPEKK_TOKEN to my shell profile. Ask me for the token value.
```
