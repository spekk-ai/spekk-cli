# Contributing to Spekk CLI

This guide covers how maintainers publish new versions of `@spekk/cli` to GemFury.

## Publishing to GemFury

### Prerequisites

You need a GemFury **publish token** (also called a "full-access" token). This is different from the read-only token used for installation.

1. **Get a publish token** from a thinknimble org admin
2. **Add the token to your shell config** (`~/.zshrc` or `~/.bashrc`):
   ```bash
   export GEMFURY_SPEKK_PUBLISH_TOKEN=your_publish_token_here
   ```
3. **Reload your shell**:
   ```bash
   source ~/.zshrc  # or ~/.bashrc
   ```

### Configure npm for Publishing

Add the publish configuration to your `~/.npmrc`:

```
@spekk:registry=https://npm.fury.io/thinknimble/
//npm.fury.io/thinknimble/:_authToken=${GEMFURY_SPEKK_PUBLISH_TOKEN}
```

Or run these commands:

```bash
npm config set @spekk:registry https://npm.fury.io/thinknimble/
npm config set //npm.fury.io/thinknimble/:_authToken "$GEMFURY_SPEKK_PUBLISH_TOKEN"
```

## Versioning

This project follows [Semantic Versioning](https://semver.org/):

- **MAJOR** (1.0.0 → 2.0.0): Breaking changes to CLI commands or spec format
- **MINOR** (1.0.0 → 1.1.0): New features, new commands, backwards-compatible additions
- **PATCH** (1.0.0 → 1.0.1): Bug fixes, documentation updates, no new features

## Pre-Publish Checklist

Before publishing a new version:

- [ ] All tests pass: `npm test`
- [ ] Spec parser still works: `spekk next` returns valid JSON
- [ ] Version in `package.json` has been bumped appropriately
- [ ] Changes are committed to git
- [ ] You're on the `main` branch (or the release branch)

## Publishing Process

### 1. Bump the Version

Update the version in `package.json`:

```bash
# For a patch release (bug fixes)
npm version patch

# For a minor release (new features)
npm version minor

# For a major release (breaking changes)
npm version major
```

This automatically:
- Updates `package.json` version
- Creates a git commit
- Creates a git tag (e.g., `v1.1.0`)

### 2. Run Tests

```bash
npm test
```

Ensure all tests pass before proceeding.

### 3. Publish to GemFury

```bash
npm publish
```

This publishes the package to the GemFury registry at `npm.fury.io/thinknimble`.

### 4. Push to Git

```bash
git push origin main --tags
```

This pushes both the commit and the version tag.

## Verifying the Publish

After publishing, verify the package is available:

```bash
# Check the package info on GemFury
npm view @spekk/cli --registry https://npm.fury.io/thinknimble/

# Or install the latest version globally
npm update -g @spekk/cli
spekk --help
```

## Troubleshooting

### "401 Unauthorized" during publish

- Verify your `GEMFURY_SPEKK_PUBLISH_TOKEN` is set: `echo $GEMFURY_SPEKK_PUBLISH_TOKEN`
- Ensure you have a **publish** token, not just a read token
- Check your `~/.npmrc` is configured correctly

### "403 Forbidden" during publish

- You may not have publish permissions in the thinknimble org
- Contact an org admin to get publish access

### Package name mismatch

Ensure `package.json` has the scoped package name:

```json
{
  "name": "@spekk/cli",
  ...
}
```

## Development Workflow

For day-to-day development:

1. Create a feature branch from `main`
2. Implement changes using the spec-driven workflow
3. Run `npm test` to verify changes
4. Create a pull request
5. After merge, follow the publishing process above from `main`
