# Spekk CLI — What's New

## Builder CLI Flags (PR #19)

The builder now supports flags for controlling scope, interaction mode, and execution behavior.

### Default behavior changed

The builder **loops continuously** by default — it picks up the next priority assertion, builds it, and moves to the next one until everything is done. Hit `Ctrl+C` to stop.

```bash
# Default: loop through all assertions until done
spekk builder
```

### New flags

| Flag | What it does |
|------|-------------|
| `--once` | Build one assertion then exit |
| `--dry-run`, `-d` | Preview what would be built without launching Claude |
| `--spec <id>`, `-s <id>` | Scope to assertions in a specific spec |
| `--assertion <id>` | Build one specific assertion (even if it's already done) |
| `--confirm`, `-c` | Prompt y/n before each build |
| `--interactive`, `-i` | Run Claude in interactive (non-headless) mode so you can collaborate |

### Try it

```bash
# See what the builder would work on next
spekk builder --dry-run --once

# Build just one assertion and stop
spekk builder --once

# Loop through only the auth spec's assertions
spekk builder --spec auth

# Build a specific assertion interactively
spekk builder --assertion login-button --interactive

# Supervised mode: loop but ask before each build
spekk builder --confirm
```

### Parser filtering

The parser also supports `--spec` and `--assertion` flags now:

```bash
# Get the next assertion from a specific spec
spekk next --spec auth

# Get a specific assertion's details
spekk next --assertion login-button
```

---

## Meeting Notes to Specs (PR #20)

The coach now has a **meeting-processing skill** that extracts todos, specs, and context updates from meeting transcripts.

### Usage

```bash
# Launch coach in meeting mode (prompts for transcript)
spekk coach meeting

# Process a transcript file directly
spekk coach meeting path/to/notes.txt
```

### What it does

1. **Extracts three categories** from your meeting transcript:
   - **Todos** — action items and follow-ups → appended to `TODOS.md`
   - **Specs** — feature discussions → proper spec files in `specs/`
   - **Context** — architectural decisions → appended to `CONTEXT.md`

2. **Proposes before creating** — shows you what it would create and waits for approval

3. **Single commit** — all outputs (todos, specs, context) committed together with a categorized commit message

### How it works

This is a **coach skill**, not a separate agent. It extends the coach's existing capabilities through the skill framework. When you run `spekk coach meeting`, the coach activates its meeting-processing mode automatically. The coach's existing knowledge of spec format, priorities, and file structure is reused — nothing is duplicated.

The skill also auto-detects meeting-related keywords in regular `spekk coach` sessions. If you mention "meeting notes", "meeting transcript", "standup notes", etc., the coach will offer to activate the skill.

---

## Quick Test Checklist

Run these to verify everything works:

```bash
# 1. Tests pass
npm test

# 2. Builder help shows all flags
spekk builder --help

# 3. Coach help shows meeting subcommand
spekk coach --help

# 4. Dry run previews next assertion (pair with --once to exit)
spekk builder --dry-run --once

# 5. Parser filtering works
spekk next --all                    # Full spec hierarchy (JSON)
spekk next --spec <some-spec-id>   # Filter to one spec

# 6. Coach meeting mode launches
spekk coach meeting                 # Should prompt for transcript
```
