---
id: parser-warnings-belong-to-the-caller
created: 2026-08-22T15:00:00Z
priority: 1
---

# The parser reports what it skipped; the caller decides who sees it

## Problem

`ParseAllSpecs` writes a warning to `os.Stderr` for every spec or assertion file it skips. The decision to print is taken inside the parse, so a caller cannot ask for the answer without the warnings, or for the warnings without the answer.

On a tree of 20 specs, each with one assertion carrying `priority: 9`:

| Command | stdout | stderr (cold index) | stderr (warm) |
|---|---|---|---|
| `spekk next` | 5 | **40** | 20 |
| `spekk list` | 4 | 20 | 20 |
| `spekk status` | 53 | 20 | 20 |
| `spekk validate` | 20 | 0 | 0 |

`spekk next` is the worst case, and it is the command a builder loop calls on every iteration. It answers in 5 lines and pays 40 to do it.

**The doubling is a side effect of printing during the parse.** `spekk next` parses the tree twice — once through `index.EnsureFresh` to build `.spekk/index.db`, once to answer — and each parse prints the full set. The count of warnings tracks the number of parses, not the number of problems. The first call in any session is cold, so an agent meets the doubled form first.

**The volume costs an agent its trust in the tool.** This is the same failure as #192, which fixed a warning that fired on healthy specs. These fire on broken ones, so they report something real, but they report it in the wrong place and at the wrong volume. An agent that cannot see the answer falls back to `ls`, `cat`, and `grep` over `specs/`, which is slower and misses specs.

## One of the eight warnings is false

`Warning: Spec %s/ has no assertions/ directory — skipping.` reports a skip that does not happen. The spec is appended to the result **before** that check, and the `continue` only ends the assertion scan. A spec directory with no `assertions/` directory parses fine and appears in `spekk status` as a normal spec with `0/0 assertions complete`, which is the correct reading of a spec somebody has drafted and not yet broken into assertions.

So the warning names a normal state, calls it a skip, and is wrong on both counts.

## Solution

The parser collects; the caller decides.

1. `ParseAllSpecs` returns its warnings on `ParseResult` and writes nothing. The doubling ends with it, because a second parse no longer emits a second copy.
2. `next`, `list`, `status`, and `show` print one line. A skipped file has vanished from the work queue, which is silent work loss and deserves a mention, but one line is enough and the detail belongs elsewhere.
3. `spekk validate` is that elsewhere, and it must cover every surviving warning, or the line's instruction to run it is a false promise.

## Scope

- In scope: the warning channel in `internal/parser`, the five callers that show output, and the coverage gap in `internal/validate`.
- Out of scope, deliberately: a severity level, a warning type or code, a `--quiet` flag, and JSON warning output. One line and one detail view answer the whole problem, and #192 already showed that deleting a noise source beats building machinery to manage it.
- Also out of scope: changing what the parser skips. Its leniency is correct for a work queue — one typo must not stall everyone — and this spec changes only who is told about it.
