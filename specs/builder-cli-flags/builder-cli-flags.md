---
id: builder-cli-flags
created: 2026-02-21T12:00:00Z
priority: 1
---

# Builder CLI Flags

## Overview

The `spekk builder` command loops continuously through assertions by default (Ctrl+C to exit). Flags control scope and interaction mode.

## Flags

| Flag | Behavior |
|------|----------|
| (none) | Loop through all assertions continuously (default) |
| `--once` | Build one assertion then exit |
| `--dry-run` | Preview what would be built, don't launch Claude |
| `--spec <id>` | Work only on assertions in this spec |
| `--assertion <id>` | Work only on this specific assertion |
| `--confirm` | Ask y/n before each build |
| `--interactive` / `-i` | Run Claude in interactive (non-headless) mode |

## Flag Combinations

- `--spec <id>` - Loop through all assertions in a spec
- `--spec <id> --once` - Build next assertion in a spec then exit
- `--confirm` - Loop through everything with confirmation prompts
- `--spec <id> --dry-run` - Preview assertions in a specific spec

## Examples

```bash
# Default: loop through all assertions
spekk builder

# Build just one and stop
spekk builder --once

# See what's next without building
spekk builder --dry-run

# Build just the login-button assertion
spekk builder --assertion login-button

# Loop through all assertions in the auth spec
spekk builder --spec auth

# Supervised autonomous mode
spekk builder --confirm
```

## Assertions

See `assertions/` for what must be true about each flag's behavior.
