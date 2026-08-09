---
icon: lucide/terminal
---

# Spekk CLI

**Spec-driven development for AI-powered teams.**

Spekk CLI lets you define _what_ must be true about your software, then uses AI agents to make it true. Write specs, not code. Let the builder implement. Let the coach guide you.

---

## How it works

``` mermaid
graph LR
  A["Write Specs"] --> B["Prioritize"];
  B --> C["Build"];
  C --> D["Validate"];
  D --> E["Ship"];
```

1. **Define specs** -- Describe what must be true (not how to build it)
2. **Prioritize** -- Order assertions by importance
3. **Build** -- AI agents implement your specs automatically
4. **Validate** -- Tests prove specs are satisfied
5. **Iterate** -- Keep specs updated as you learn

---

## Quick start

Install with one command (macOS / Linux — installs to user-owned `~/.local/bin`, no sudo needed):

```bash
curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh
```

Then start building:

```bash
spekk coach       # Create a spec
spekk next        # See what's next
spekk builder     # Build it
```

[Get started](getting-started.md){ .md-button .md-button--primary }
[CLI Reference](cli-reference.md){ .md-button }

---

## Core commands

| Command | What it does |
|---------|-------------|
| `spekk next` | Get the next priority assertion to work on |
| `spekk builder` | Launch the builder agent to implement specs |
| `spekk coach` | Launch the coach agent to create specs |
| `spekk status` | Comprehensive overview of all specs |
| `spekk show` | Interactive web-based spec explorer |
| `spekk serve` | WebSocket server for browser extension |
| `spekk sandbox` | Manage cloud sandbox environments |

---

## The agents

=== "Builder"

    Automates implementation of your specifications. Reads assertions, writes tests, implements features, and commits changes.

    ```bash
    spekk builder --once     # Build one assertion
    spekk builder            # Loop through all
    spekk builder --dry-run  # Preview without executing
    ```

=== "Coach"

    Helps you write well-formed specs. Takes vague ideas and turns them into atomic, testable assertions.

    ```bash
    spekk coach                    # Interactive spec creation
    spekk coach meeting notes.txt  # Process meeting transcript
    spekk coach coordinate         # Plan dependencies
    ```

=== "Observer"

    Finds drift between your specs and code. Detects when implementation changes but specs don't update (or vice versa).

    ```bash
    spekk observer
    ```

---

## Philosophy

!!! quote "Specs are the source of truth"

    Code is the implementation of specs. Tests prove specs are satisfied.

- **Write specs first** -- Define what must be true
- **Let AI implement** -- Focus on the _what_, not the _how_
- **Validate with tests** -- Prove correctness automatically
- **Iterate continuously** -- Specs evolve with your understanding
