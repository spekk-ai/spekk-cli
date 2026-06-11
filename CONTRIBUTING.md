# Contributing to Spekk CLI

## Pull Requests

PRs are welcome but reviewed at the maintainers' discretion. We are a small team and may close PRs without detailed explanation. If you're planning significant work, open an issue first to discuss it — that saves everyone time.

## Development Setup

### Prerequisites

- **Go** 1.23+ ([install](https://go.dev/dl/))
- **Claude CLI** (for agent features)
- **Git**

### Building

```bash
go build ./cmd/spekk
```

The binary is output to the current directory as `spekk`.

### Testing

```bash
# Run all tests
go test ./...

# Run a specific package
go test ./internal/parser/ -v

# Run with race detector
go test -race ./...
```

## Versioning

This project follows [Semantic Versioning](https://semver.org/):

- **MAJOR** (1.0.0 -> 2.0.0): Breaking changes to CLI commands or spec format
- **MINOR** (1.0.0 -> 1.1.0): New features, new commands, backwards-compatible additions
- **PATCH** (1.0.0 -> 1.0.1): Bug fixes, documentation updates, no new features

## Publishing a Release

Releases are published via GitHub Releases. The CI workflow automatically builds cross-platform binaries when a tag is pushed.

### 1. Tag the Release

```bash
git tag v1.3.0
git push origin v1.3.0
```

### 2. CI Builds Binaries

The GitHub Actions workflow automatically:
- Runs all tests
- Cross-compiles binaries for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
- Creates a GitHub Release with the binaries attached

### 3. Verify

Check the [Releases page](https://github.com/spekk-ai/spekk-cli/releases) to confirm binaries are available.

## Development Workflow

1. Create a feature branch from `main`
2. Implement changes using the spec-driven workflow
3. Run `go test ./...` to verify changes
4. Create a pull request
5. After merge, tag and push a release from `main`

## Project Structure

```
cmd/spekk/          # CLI entry point (main.go)
internal/
  agent/            # Agent launchers (coach, builder, observer, loops)
  cli/              # Flag parsing, prompt/skill resolution
  parser/           # Spec parser
  sandbox/          # Cloud sandbox management
  serve/            # WebSocket server
  show/             # Spec explorer web UI
  status/           # Status overview display
specs/              # Specification files
```
