---
icon: lucide/terminal
---

# Spekk CLI

**Spec-driven development for teams that work with AI agents.**

With Spekk you write what must be true about your software, and AI agents make it true. You write specs. The builder implements them. The coach helps you write them. The observer tells you when the code and the specs drift apart.

## How it works

``` mermaid
graph LR
  A["Write Specs"] --> B["Prioritize"];
  B --> C["Build"];
  C --> D["Validate"];
  D --> E["Ship"];
```

1. **Write specs.** Say what must be true, not how to build it.
2. **Prioritize.** Order the assertions by importance.
3. **Build.** The builder agent implements the specs.
4. **Validate.** Tests prove the specs are satisfied.
5. **Iterate.** Keep the specs current as you learn.

## Quick start

Install with one command on macOS or Linux. The script installs to `~/.local/bin`, which you own, so no sudo is needed:

```bash
curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh
```

Then:

```bash
spekk init        # Create specs/
spekk coach       # Write a spec
spekk next        # See what is next
spekk builder     # Build it
```

[Get started](getting-started.md){ .md-button .md-button--primary }
[CLI Reference](cli-reference.md){ .md-button }

## Core commands

| Command | What it does |
|---------|-------------|
| `spekk next` | Print the next assertion to work on |
| `spekk builder` | Launch the builder agent to implement specs |
| `spekk coach` | Launch the coach agent to write specs |
| `spekk observer` | Launch the observer agent to find drift |
| `spekk status` | Overview of every spec and assertion |
| `spekk validate` | Check the spec tree, exit 1 on a fault |
| `spekk show` | Spec explorer in the browser |
| `spekk sandbox` | Manage sandboxes for remote agents |

## The agents

=== "Builder"

    Implements your specs. It reads an assertion, writes tests, implements the change, and commits.

    ```bash
    spekk builder --once     # Build one assertion
    spekk builder            # Build assertions until you stop it
    spekk builder --dry-run  # Print the next assertion, build nothing
    ```

=== "Coach"

    Helps you write well-formed specs. It turns an idea into small, testable assertions.

    ```bash
    spekk coach                    # Interactive spec creation
    spekk coach meeting notes.txt  # Turn a meeting transcript into specs
    spekk coach coordinate         # Plan dependencies and branches
    ```

=== "Observer"

    Finds drift between your specs and your code: a change to the code that the specs do not record, and a change to the specs that the code does not implement.

    ```bash
    spekk observer                 # Run one scan, file one observation
    spekk observer install-cron    # Run it once a day
    ```

## Philosophy

!!! quote "Specs are the source of truth"

    The code implements the specs. The tests prove the specs are satisfied.

- **Write specs first.** Say what must be true.
- **Let the agents implement.** Focus on the what, not the how.
- **Validate with tests.** Prove the specs are satisfied.
- **Iterate.** The specs change as your understanding changes.
