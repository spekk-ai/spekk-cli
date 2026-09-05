---
icon: lucide/rocket
---

# Getting Started

Install spekk, write one spec, and build it.

## Prerequisites

- **Claude Code CLI**, for the `spekk coach`, `spekk builder`, and `spekk observer` launchers. You do not need it when you run the agents from another assistant.
- **Git**, for branch-aware parsing and the commits the agents make.
- **Go** 1.25 or later, only to build from source.

## Installation

### Install script

The script detects your platform, installs to `~/.local/bin` (no sudo, for the install or a later update), and warns you when that directory is not on your `PATH`:

```bash
curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh
```

See the [install guide](install.md) for manual downloads, Windows, and the script's options.

### From source

```bash
go install github.com/spekk-ai/spekk-cli/cmd/spekk@latest
```

### Verify

```bash
spekk version
```

### Update

```bash
spekk update           # Install the latest release
spekk update --check   # See what is available, install nothing
```

A build from source updates with `go install` again.

## Your first spec

### Set up the project

```bash
cd your-project
spekk init
```

This creates `specs/` at the git root, with a README that describes the format.

### Write a spec with the coach

```bash
spekk coach
```

Tell the coach what you want to build:

```
> "I need to add user authentication to my app"
```

The coach asks questions, then writes a spec with clear success criteria, small assertions, and priorities.

### Look at the specs

```bash
spekk next      # What is next?
spekk status    # Overview of every spec and assertion
spekk show      # Spec explorer in the browser
spekk validate  # Check the tree for faults
```

### Build it

```bash
spekk builder --once   # Build the next assertion
spekk builder          # Build assertions until you stop it
```

## Directory structure

After the coach has run, the project has:

```
your-project/
├── specs/
│   ├── README.md                    # Written by spekk init
│   ├── authentication/
│   │   ├── authentication.md        # The spec
│   │   └── assertions/
│   │       ├── password-hashing.md  # One assertion
│   │       └── session-tokens.md    # One assertion
│   └── another-feature/
│       ├── another-feature.md
│       └── assertions/
│           └── ...
├── .spekk/           # Prompt and skill customizations, and derived files such as the index
├── TODOS.md          # Action items, from the meeting skill
└── CONTEXT.md        # Decisions, from the meeting skill
```

## Common workflows

### A new feature

```bash
spekk coach                           # 1. Write the spec
spekk coach coordinate                # 2. Plan dependencies and branches
git checkout -b feature/authentication  # 3. Make the feature branch
spekk builder --once                  # 4. Build the first assertion
```

### Meeting notes

```bash
spekk coach meeting notes.txt   # Turn the transcript into specs, todos, and context
cat TODOS.md                    # Review the action items
spekk next --all                # Review the new specs
```

### Sprint planning

```bash
spekk status            # What is ready?
spekk coach coordinate  # Plan the work across branches
spekk builder           # Start the build loop
```

## Next steps

- [Concepts](concepts.md): specs, assertions, and priorities
- [CLI Reference](cli-reference.md): every command and flag
- [Skills](coach-skills.md): meeting notes, coordination, and your own skills
- [Configuration](configuration.md): agent prompts, environment variables, and the suppression file
- [Validation in CI](ci.md): run `spekk validate` in a pre-commit hook and in CI
