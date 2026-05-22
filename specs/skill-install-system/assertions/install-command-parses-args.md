---
id: install-command-parses-args
parent: skill-install-system
created: 2026-05-22T12:00:00Z
priority: 1
status: not_started
---

# Install Command Parses Agent, Skill, and Flags

## Description

`spekk install` accepts a positional `<agent>`, an optional positional `<skill>`, and the flags `--global`, `--local`, `--source <URL>`, `--force`, and `--list <agent>`. Flag parsing reuses `internal/cli.ParseFlags` to match the rest of the codebase.

## Success Criteria

- `spekk install coach meeting-notes` parses to: agent=`coach`, skill=`meeting-notes`, scope=local, source="", force=false
- `spekk install builder my-skill --global` sets scope=global
- `spekk install builder my-skill --local` sets scope=local (explicit form)
- `spekk install coach foo --source https://x.com/foo.md` populates the source field
- `spekk install coach foo --force` sets force=true
- Passing both `--global` and `--local` exits non-zero with an error explaining they're mutually exclusive
- Omitting `<agent>` prints usage and exits non-zero
- Passing an unknown agent (e.g. `spekk install bogus foo`) exits non-zero with a message listing the valid agents
- `spekk install --help` prints usage describing all flags
