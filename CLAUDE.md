# Claude Code Configuration

This project is the CLI for spec-driven development with coach and builder agents.

## Agent Prompts

**Builder Agent:** `specs/builder-agent/builder.prompt.md`
- Use this for implementing assertions and making specs true

**Coach Agent:** `specs/coach-agent/coach.prompt.md`
- Use this for refining user requests into well-formed specs

## CLI Usage

This CLI tool allows you to run spec-driven development workflows externally:

- `spekk coach` - Start the coach agent for spec creation
- `spekk builder` - Start the builder agent for implementation
- `spekk next` - Get the next priority assertion to work on

## Project Structure

- `cmd/spekk/` - CLI entry point
- `internal/` - Core Go packages (parser, agent, cli, show, serve, sandbox, status)
- `specs/` - Specification files organized by feature

## Development

- `go build ./cmd/spekk` - Build the binary
- `go test ./...` - Run all tests
