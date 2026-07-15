# Migrating to the new Spekk CLI (Go binary)

The spekk CLI has been rewritten from Node.js/npm to a single Go binary. This guide covers how to switch over.

---

## 1. Remove the old npm version

```bash
# Uninstall the global npm package
npm uninstall -g spekk-cli

# Verify it's gone
which spekk  # should return nothing or "not found"
```

If you installed it locally in a project, also remove it from your `package.json`:

```bash
npm uninstall spekk-cli
```

## 2. Install the new Go binary

Install the latest release with the one-line installer, which downloads the
right binary for your platform from GitHub Releases and places it on your PATH:

```bash
curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh

# Verify
spekk help
```

By default it installs to `~/.local/bin`. On Windows, download
`spekk-windows-amd64.exe` (or the `arm64` variant) from the
[latest release](https://github.com/spekk-ai/spekk-cli/releases/latest) and
place it on your PATH. Once installed, `spekk update` self-updates to the
latest release.

> **Note:** The binary is a standalone executable — no npm, no Node.js, no dependencies required.

---

## 3. What changed

### Single binary, no dependencies

The CLI is now a compiled Go binary with all agent prompts and seed skills embedded. There is no `node_modules`, no `package.json`, no runtime dependencies. You download one file and it works.

### Skill install system

You can now install community skills from the official registry:

```bash
spekk install coach meeting-notes          # install to current project
spekk install coach meeting-notes --global # install for all projects
spekk install --list coach                 # see available skills
spekk uninstall coach meeting-notes        # remove a skill
spekk skill list coach                     # see all skills (local + global + embedded)
```

**Important:** There are no packages in the registry yet — the infrastructure is in place but the community skill library is empty for now. The embedded seed skills (shipped with the binary) still work as before.

Scopes:
- **Local** (default): `.spekk/skills/<agent>/` in your project
- **Global** (`--global`): `~/.spekk/skills/<agent>/` for all projects

### Observer agent now supports skills

The observer agent previously had no skill support. It now uses the same layered skill discovery as coach and builder:

```bash
spekk observer coverage-gap     # run observer with a specific skill
spekk observer --interval 60    # default monitoring mode (unchanged)
```

A seed skill (`coverage-gap`) ships embedded in the binary. It scans for exported code that has no backing spec.

### Skill resolution order (unchanged)

1. **Local** — `.spekk/skills/<agent>/` (project-specific, highest priority)
2. **Global** — `~/.spekk/skills/<agent>/` (user-wide)
3. **Embedded** — shipped with the binary (lowest priority)

### All existing commands still work

```
spekk coach       # launch coach agent
spekk builder     # launch builder agent
spekk observer    # launch observer agent
spekk show        # spec explorer
spekk status      # overview of all specs
spekk serve       # websocket server for browser extension
spekk sandbox     # cloud sandbox management
```

---

## 4. Quick checklist

- [ ] Run `npm uninstall -g spekk-cli`
- [ ] Install the Go binary: `curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh`
- [ ] Run `spekk help` to verify
- [ ] (Optional) Run `spekk update` to confirm self-update works
