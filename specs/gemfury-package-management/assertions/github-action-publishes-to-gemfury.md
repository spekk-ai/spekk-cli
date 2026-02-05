---
id: github-action-publishes-to-gemfury
parent: gemfury-package-management
created: 2026-02-05T18:00:00Z
priority: 1
status: done
---

# GitHub Action Publishes to GemFury

## What Must Be True

A GitHub Action workflow automates publishing the `@spekk/cli` package to GemFury. The workflow is manually triggered, runs tests before publishing, and uses proper npm packaging configuration.

## Success Criteria

### Workflow File
- `.github/workflows/publish.yml` exists
- Triggered via `workflow_dispatch` (manual)
- Uses Node.js 20
- Runs `npm ci` to install dependencies
- Runs `npm test` before publishing
- Packs the tarball with `npm pack`
- Pushes to GemFury using `GEMFURY_TOKEN` secret via curl

### NPM Ignore
- `.npmignore` file exists
- Excludes `__tests__/` directories
- Excludes `*.test.js` and `*.test.sh` files
- Excludes `src/test-runner.js` and `src/test-runner-mocked.js`
- Excludes `.tmp/` directory

### NPM Registry Config
- `.npmrc` file exists
- Configures `@spekk` scope to use `https://npm.fury.io/thinknimble/`

### Package.json Updates
- Name is scoped: `@spekk/cli`
- `files` array specifies included files:
  - `bin/`
  - `src/` (excluding test files)
  - `specs/builder-agent/`
  - `specs/coach-agent/`
- `publishConfig.registry` points to `https://npm.fury.io/thinknimble/`
- `prepublishOnly` script runs `npm test`

## Validation

- GitHub Action runs successfully when triggered
- Published package contains only production files (no tests)
- Package installs correctly from GemFury registry
