---
id: readme-contains-installation-instructions
parent: gemfury-package-management
created: 2026-02-05T12:00:00Z
priority: 1
status: done
---

# README Contains GemFury Installation Instructions

## What Must Be True

The README.md file contains a clear "Installation" section that explains how to install @spekk/cli from the GemFury private registry.

## Success Criteria

- [ ] README has an "Installation" section (before "Quick Start" or development instructions)
- [ ] Instructions cover getting a GemFury token from a team lead
- [ ] Instructions explain adding the token to shell config (~/.zshrc or ~/.bashrc)
- [ ] Instructions show how to configure npm to use the GemFury registry for @spekk scope
- [ ] Instructions include the install command: `npm install -g @spekk/cli`
- [ ] Instructions include verification step: `spekk --help`
- [ ] Instructions include how to update: `npm update -g @spekk/cli`

## Reference Content

The installation section should include these steps:

### 1. Get a Gemfury token
Ask a team lead for a Gemfury **read** (or full access) token for the `thinknimble` org.

### 2. Add the token to your shell
```bash
# Add to ~/.zshrc or ~/.bashrc
export GEMFURY_SPEKK_TOKEN=your_token_here
```
Then reload: `source ~/.zshrc` (or `~/.bashrc`)

### 3. Configure npm
Add to global `~/.npmrc`:
```
@spekk:registry=https://npm.fury.io/thinknimble/
//npm.fury.io/thinknimble/:_authToken=${GEMFURY_SPEKK_TOKEN}
```

Or run:
```bash
npm config set @spekk:registry https://npm.fury.io/thinknimble/
npm config set //npm.fury.io/thinknimble/:_authToken "$GEMFURY_SPEKK_TOKEN"
```

### 4. Install
```bash
npm install -g @spekk/cli
```

### 5. Verify
```bash
spekk --help
```

### Updating
```bash
npm update -g @spekk/cli
```
