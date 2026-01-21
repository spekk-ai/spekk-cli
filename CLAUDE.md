# Claude Code Configuration

This project is the CLI for spec-driven development with coach and builder agents.

## Agent Prompts

**Builder Agent:** `specs/builder-agent/builder-agent.prompt.md`
- Use this for implementing assertions and making specs true

**Coach Agent:** `specs/coach-agent/coach-agent.prompt.md`
- Use this for refining user requests into well-formed specs

## CLI Usage

This CLI tool allows you to run spec-driven development workflows externally:

- `npm run coach` - Start the coach agent for spec creation
- `npm run builder` - Start the builder agent for implementation
- `npm run next` - Get the next priority assertion to work on

## Project Structure

- `app/` - Core application logic for parser, coach, and builder
- `bin/` - CLI entry points 
- `specs/` - Specification files organized by feature