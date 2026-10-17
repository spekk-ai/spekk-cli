---
id: next-says-why-nothing-is-eligible
created: 2026-10-17T00:00:00Z
priority: 2
---

# `spekk next` says why no assertion is eligible

## Problem

`spekk next` prints `"type": "complete"` and "All specifications are complete" when it finds no eligible assertion. That is false when open assertions remain but none is eligible: all of them are on a different branch, blocked by a dependency, locked by a live builder, or draft. An assertion with no `branch` field belongs to `main`, so on a feature branch a new assertion without the field is not eligible. A person or a builder loop that reads "complete" thinks that the work is done (issue #271).

## Solution

`FindNextAssertion` and a new count function use one rule for each reason. When no assertion is eligible and open assertions remain, `spekk next` prints `"type": "none_eligible"` with the number of open assertions, the number for each reason, and a message that names the reasons. `"type": "complete"` stays for the case where no assertion is open. `spekk status` prints the same message in place of "All specifications complete!". The builder loops accept the new type, print its message, and wait (or exit 0 with `--once`), as they do for `complete`.

## What this deliberately does not do

- **No change to which assertion is eligible.** Only the output when none is eligible changes.
- **No list of the assertion ids.** The counts and the `--all-branches` hint are enough to find them with `spekk list`.
