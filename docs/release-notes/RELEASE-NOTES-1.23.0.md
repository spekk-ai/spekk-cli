# 1.23.0 — The Answer, Not the Warnings

Two warnings had grown loud enough to hide the output they came with. One fired on healthy specs and is deleted. The other fires on broken ones and now reports in a single line. A command that answers a question gives you the answer; `spekk validate` gives you the detail.

Read [Upgrading](#upgrading) before you raise your pin. `spekk validate` is stricter in this release, and a repository that passes today can fail after the upgrade.

## The `branch` guard checked a convention, not a branch

The guard matched a `branch` value against fourteen fixed words and never looked at git. It was wrong in both directions.

It stayed quiet where the harm was. `feat/retry-billing-webhok` passed, because `feat/` is on the list — and it names no branch, so `spekk next` filtered that assertion out of the queue and would have done so for ever, with no message.

It was loud where there was none. A team that puts a developer name first, `dana/apx-12-retry-billing-webhook`, got a warning on every assertion carrying the field. Git accepts the name, and the only thing that reads the field compares it to the current branch as a plain string, so the shape changes nothing.

The word list is gone, along with the pattern built from it and the warning it printed. The errors that remain reject only names git itself refuses: bad characters, `..`, a trailing `.`, a `.lock` suffix, a leading or trailing `/`.

In its place, `spekk validate` reads the refs. An assertion that is neither `done` nor `draft` and names a branch that no ref matches is reported once per distinct value:

```
Warning: branch "feat/retry-billing-webhok" does not exist (3 assertions not done). spekk next cannot reach that work.
```

A `done` assertion on a deleted branch is merged work, and stays silent. On a tree of 61 assertions across three team-convention branch values, stderr on `spekk next`, `spekk list`, and `spekk status` went from 183 lines to none.

There is no configuration for this, and none is needed. Once the check reads a real ref, no naming preference is left to configure.

## Skipped files report once, not once each

The parser is lenient on purpose: a file it cannot parse is skipped, so one typo never stalls the whole queue. It also printed a warning per skipped file, from inside the parse, which meant the volume tracked how many times the tree was parsed rather than how many problems it held. `spekk next` parses twice — once to build the index, once to answer — so it printed everything twice, and the first call in any session is the cold one.

Each command now prints one line:

```
Warning: 20 spec files skipped and missing from the queue. Run "spekk validate" for detail.
```

On a tree of 20 malformed assertions:

| Command | before | after |
|---|---|---|
| `spekk next` | 40 | 1 |
| `spekk list` | 20 | 1 |
| `spekk status` | 20 | 1 |

Cold and warm runs now agree, because the doubling is gone. The line goes to stderr, so `--json`, `--csv`, and `--tsv` stay machine-readable.

One warning was deleted rather than shortened. `Spec X/ has no assertions/ directory — skipping` reported a skip that never happened: such a spec parses correctly and appears in `spekk status` as `0/0 assertions complete`, which is the right reading of a spec nobody has broken into assertions yet.

## A lock means a builder holds it now

`spekk validate` demanded a `locked-by` on every `in_progress` assertion. No coach could satisfy that. A lock names a builder session, and no command mints one, so a coach following its own instruction to move an edited assertion to `in_progress` wrote a tree the validator rejected. The only way past it was an invented lock, which cannot be told apart from the lock a crashed builder left behind.

The code already disagreed with the rule: `spekk next` treats an unlocked `in_progress` assertion as free work. `validate` now allows it too. The reverse rule stands — a `done`, `failed`, `not_started`, or `draft` assertion carrying a lock is still a failure.

A stale lock is now reported, which the old shape made impossible: it flagged a missing lock and a lock on a settled assertion, but never one months old, because that shape was legal.

The coach prompt writes `not_started` when it edits a `done` or `failed` assertion. That returns it to the queue, which was always the intent, and it needs no lock.

## Installs land where you mean

`spekk install` decides that a file at a managed path belongs to spekk from the path alone, and brings it to the current content, keeping whatever was there in a `.bak`. Ownership used to be re-derived from the file body, so rewording a shim left every copy in the field stale. A symlink at a managed path, or a file spekk cannot read, is reported rather than written through: two tools own that path, and only you can say which.

`spekk install --target <tool> --project` and the `spekk update` report now work on the repository rather than on the directory you happen to stand in. `--project` joined the working directory, so running it in `repo/src` wrote a second install to `repo/src/.claude` instead of updating `repo/.claude`, and the report read the same way — a stale project file one level up went unmentioned, and the remedy it printed made the second install rather than fixing the first.

## Also

- `spekk list --json` and `spekk next --all` carry the `branch` field on spec and assertion objects. The parser always read it; only the flat and hierarchy views dropped it, so an agent enumerating work needed a second call per record to learn the branch.
- The coach and builder prompts recommend `spekk query` for finding an assertion, in place of piping `spekk list --json` into `grep`. The JSON is indented, so grep returns a bare `"title"` line with no id and no file, and it matches a keyword in a path as readily as one in a title. Neither source holds an assertion body, so both see the same fields and SQL is the better way to filter them. `grep -rl` stays the tool for searching spec prose.

## Upgrading

**`spekk validate` gained three failure conditions, and a repository that passes today can fail after this upgrade.** Each one loses work silently, which is why it is a failure rather than a warning:

- A spec directory that holds assertion files but no main spec file. The parser drops the whole directory, so every assertion in it is missing from the queue.
- A path named `assertions` that is not a directory.
- An `assertions/` directory that cannot be read.

Run `spekk validate` once against the new binary before you raise a pinned `SPEKK_VERSION` or a pre-commit `rev`. This is the case the pinning advice in [Validation in CI and pre-commit](ci.md) exists for.

A spec directory with **no** `assertions/` directory still passes. It is not a fault.

Nothing else here fails a tree that passed before. The branch guard only became more permissive, and both new reports are warnings that leave the exit code alone.
