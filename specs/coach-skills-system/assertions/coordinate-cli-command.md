---
id: coordinate-cli-command
parent: coach-skills-system
created: 2026-02-25T15:58:00Z
priority: 1
status: done
---

# Coach Accepts "coordinate" Subcommand

The coach CLI accepts `spekk coach coordinate` to activate the coordinator skill.

## What Must Be True

### CLI Command
- User runs: `spekk coach coordinate`
- Coach launches with coordinator skill active
- Follows same pattern as `spekk coach meeting`

### Activation Message
- Coach receives skill activation message
- Message tells coach to activate coordinator skill immediately
- No trigger detection needed (explicit invocation)

### Help Documentation
- `spekk coach --help` shows `coordinate` subcommand
- Help text describes: "Analyze work and create dependency-aware plan"

## Implementation

In `src/coach/cli.js`:
- Change `if (subcommand === 'coordinator')` to `if (subcommand === 'coordinate')`
- Update activation message to reference correct command
- Update help text

## Validation

```bash
spekk coach coordinate
# Should activate coordinator skill

spekk coach --help
# Should list "coordinate" subcommand
```
