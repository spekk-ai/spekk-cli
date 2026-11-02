---
id: constraints-command
created: 2026-11-02T21:30:00Z
priority: 2
---

# `spekk constraints <file>` - the Properties an Edit Can Break

## Problem

An agent reads the relevant properties first, and then reads only the code it will edit. A property replaces the code that the agent will not edit but can break. For this, the agent needs one command that answers: before I edit this file, which properties anchor on it, and can I trust them?

## Decisions

- The command lists the properties that anchor on the file, with their truth at HEAD (William, 2026-11-01).
- A `PreToolUse` hook that runs `spekk constraints` before an edit is later work.

## Open choices

- **Anchors only.** A property is in the list when it has a comment anchor in the file or a front-matter anchor that names the file. A marker in a test file does not put its property in the list of that test file.
- **Output.** The default is a table like `spekk list`, with `ID`, `STATUS`, `TRUTH`, `LINES` and `TITLE`. `--json` prints an array. The other list formats are not added.
- **No match.** A file with no anchors prints nothing on stdout, `no properties anchor on <path>` on stderr, and exits 0. A file that does not exist yet is not an error, because an agent can ask before it creates a file.

## Assertions

See `assertions/` for what must be true.
