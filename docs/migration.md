# Migrating to the new Spekk CLI (Go binary)

The spekk CLI was rewritten from Node.js and npm to a single Go binary in 1.5.0. This guide covers the switch.

## 1. Remove the old npm version

```bash
npm uninstall -g spekk-cli   # Remove the global package
which spekk                  # Must print nothing, or "not found"
```

When you installed it in a project, remove it from `package.json` too:

```bash
npm uninstall spekk-cli
```

## 2. Install the Go binary

The install script downloads the binary for your platform from GitHub Releases and puts it on your `PATH`:

```bash
curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh
spekk help   # Verify
```

The default location is `~/.local/bin`. On Windows, download `spekk-windows-amd64.exe` (or the `arm64` variant) from the [latest release](https://github.com/spekk-ai/spekk-cli/releases/latest) and put it on your `PATH`. After the install, `spekk update` updates to the latest release.

The binary is one file. It needs no npm, no Node.js, and no other dependency.

## 3. What changed

### One binary

The CLI is a compiled Go binary with every agent prompt and every seed skill inside it. There is no `node_modules`, no `package.json`, and no runtime dependency.

### Skill installation

You can install skills from the registry:

```bash
spekk install coach meeting-notes          # Into the current project
spekk install coach meeting-notes --global # For every project
spekk install --list coach                 # What the registry has
spekk uninstall coach meeting-notes        # Remove a skill
spekk skill list coach                     # Every skill: local, global, and embedded
```

Scopes:

- **Local** (default): `.spekk/skills/<agent>/` in the project
- **Global** (`--global`): `~/.config/spekk/skills/<agent>/`, for every project

### The observer supports skills

The observer uses the same layered skill discovery as the coach and the builder:

```bash
spekk observer coverage-gap   # Run one skill
spekk observer                # Run one scan
```

Three skills ship in the binary: `coverage-gap`, `prune`, and `consolidate`. See [Skills](coach-skills.md#built-in-observer-skills).

### Skill resolution order

1. **Local**: `.spekk/skills/<agent>/`, the project. Highest priority
2. **Global**: `~/.config/spekk/skills/<agent>/`, your user directory
3. **Embedded**: the skills in the binary. Lowest priority

### The global directory moved

1.7.0 moved the global directory from `~/.spekk` to `~/.config/spekk` (or `$XDG_CONFIG_HOME/spekk`). The first spekk command after the upgrade moves an old `~/.spekk` for you. See [Configuration](configuration.md).

### The commands you know still work

```
spekk coach       # Launch the coach agent
spekk builder     # Launch the builder agent
spekk observer    # Launch the observer agent
spekk show        # Spec explorer
spekk status      # Overview of every spec
spekk serve       # WebSocket server for the browser extension
spekk sandbox     # Sandbox management
```

## 4. Checklist

- [ ] Run `npm uninstall -g spekk-cli`
- [ ] Install the Go binary: `curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh`
- [ ] Run `spekk help` to verify
- [ ] Optional: run `spekk update --check` to confirm that self-update works
