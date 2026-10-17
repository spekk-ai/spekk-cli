---
id: next-reports-none-eligible
parent: next-says-why-nothing-is-eligible
created: 2026-10-17T00:00:00Z
priority: 2
status: not_started
---

# `spekk next` reports `none_eligible` with the count and the reasons

## Success Criteria

- When no assertion is open (all are `done`), `spekk next` prints `"type": "complete"` as before.
- When open assertions remain and none is eligible, `spekk next` prints `"type": "none_eligible"`, `"remaining"` (the number of open assertions in scope), `"reasons"` (a count for each of `other_branch`, `blocked`, `locked`, `draft`, with only the reasons that occur), and a `"message"` that names the counts. It exits 0.
- The message for `other_branch` says that an assertion with no `branch` field belongs to `main`, and tells the user to run `spekk next --all-branches`.
- With `--spec <id>`, only the assertions of that spec are counted.
- `spekk status` prints the same message in place of "All specifications complete!" when open assertions remain.
- `spekk builder` and the builder loop treat `none_eligible` as "no work now": they print the message and wait, or exit 0 with `--once`. They do not print "All assertions completed".
- `docs/cli-reference.md` describes the new type.
- Live check: on a feature branch, add an assertion with no `branch` field and run `spekk next`. The output is `none_eligible` with `other_branch: 1` or more.
