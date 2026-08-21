---
id: branch-field-typo-guard
created: 2026-08-21T17:35:19Z
priority: 1
---

# The `branch` field is guarded against a typo, not against a naming convention

## Problem

`validateBranch` in `internal/parser/parser.go` warns when a `branch` value does not match `main`, `master`, `develop`, or `<type>/<name>` for one of 14 fixed words. The rule is wrong in both directions.

**It is quiet where the harm is.** The guard never looks at git. It matches a string against a word list. So `feat/retry-billing-webhok` passes silently, although it names no branch. `internal/parser/parser.go:928` then drops that assertion from the work queue, because its branch does not equal the current branch and never will. The assertion becomes invisible to `spekk next` for ever, with no message.

**It is loud where there is no harm.** A team that puts a developer name first (`dana/apx-12-retry-billing-webhook`) gets a warning on every assertion. Git accepts the name. The only consumer of the field compares it to the current branch as a plain string, so the shape changes nothing. GitHub issue #192 reports 61 warnings from 3 distinct values on one project, at 3 lines each: 183 lines of stderr on every command that reads the spec tree, including `spekk next`, which a builder loop calls on each iteration.

The commit that last shaped the list (29568fe) states the real intent: the warning "reports a value in neither convention, such as the bare name `temporary-target` or a spelling error. Such a value usually names no branch." The intent is a typo catch. A word list is a poor proxy for it.

## Solution

Delete the convention rule, and check the thing the rule was a proxy for: does the value name a branch that exists?

A missing ref alone is not evidence of a typo, because a merged branch is deleted and that is normal. The second signal is nearness: a typo is by definition almost a real name. So the check reports a value that names no ref **and** sits within a small edit distance of a ref that does exist, and it names that ref.

## Scope

- In scope: removal of the word list and its warning; a typo report in `spekk validate`.
- Out of scope, and deliberately so: a `.spekk/config.yml` branch rule (issue #40). Once the guard checks a real ref instead of a naming convention, no team preference is left to configure. Also out of scope: a general parser warnings framework, a `--quiet` flag, and warning grouping anywhere but the one report below. Deleting the noise source beats building machinery to manage it.

## Design decisions to sanity-check

- **The report lives in `validate`, not in the parser.** `validateBranch` runs once per file and is pure. A git call there would run hundreds of times per command. `validate` is the command whose job is to report problems, and `builder-runs-validate` already makes the builder loop run it before it commits.
- **`next`, `list`, `status`, and `show` gain no new output.** Their job is to answer, not to report. After the word list is gone they print no branch warnings at all.
- **Reuse `crossbranch.DiscoverAllBranches("")`** for the ref set. It already returns the deduplicated union of local heads and remote-tracking refs by logical name, through the read-only git chokepoint.
