# 1.23.0 — Quieter Output and a Real Branch Check

Two warnings made the output difficult to read. The first warning fired on correct specs, and this release deletes it. The second warning fires on files that the parser skips, and this release makes it one line.

A command that answers a question now gives you the answer. `spekk validate` gives you the detail.

Read [Upgrading](#upgrading) before you change a version pin. `spekk validate` is more strict in this release. A repository that passes now can fail after you upgrade.

## The branch guard did not read git

The guard compared a `branch` value with 14 fixed words. It did not read the git refs. The result was wrong in two directions.

The guard accepted a branch that does not exist. For example, `feat/retry-billing-webhok` starts with `feat/`, so the guard accepted it. No branch has that name. `spekk next` compares the field with the current branch, so it removed that assertion from the queue and gave no message.

The guard also refused a branch that git accepts. Some teams put a developer name first, such as `dana/apx-12-retry-billing-webhook`. Each assertion with such a value got a warning. Git accepts the name. Only one part of spekk reads this field, and it compares the value with the current branch as text. Thus the shape of the name has no effect.

This release deletes the word list, the pattern, and the warning. The errors that stay refuse only the names that git also refuses. These are incorrect characters, `..`, a final `.`, a `.lock` suffix, and an initial or final `/`.

`spekk validate` now reads the refs. It reports an assertion when two conditions are true: the status is not `done` and not `draft`, and no ref matches the branch value. It reports each different value one time:

```
Warning: branch "feat/retry-billing-webhok" does not exist (3 assertions not done). spekk next cannot reach that work.
```

A `done` assertion on a deleted branch is merged work, and `validate` does not report it.

A test tree had 61 assertions and three team branch values. Before this release, `spekk next`, `spekk list`, and `spekk status` each wrote 183 lines to stderr. They now write none.

There is no configuration for this check, and you do not need one. The check reads a real ref, so no naming preference stays.

## Skipped files give one report

The parser is permissive on purpose. It skips a file that it cannot read, so one error does not stop the queue.

The parser also wrote one warning for each skipped file during the parse. Thus the number of lines showed how many times spekk parsed the tree. `spekk next` parses the tree two times: one time to build the index, and one time to answer. Thus it wrote each warning two times. The first call in a session always builds the index.

Each command now writes one line:

```
Warning: 20 spec files skipped and missing from the queue. Run "spekk validate" for detail.
```

The table below shows a tree with 20 incorrect assertions:

| Command | Lines before | Lines after |
|---|---|---|
| `spekk next` | 40 | 1 |
| `spekk list` | 20 | 1 |
| `spekk status` | 20 | 1 |

A first call and a later call now give the same number of lines, because spekk no longer writes each warning two times. The line goes to stderr. Thus `--json`, `--csv`, and `--tsv` stay machine-readable.

This release deletes one warning. The message `Spec X/ has no assertions/ directory — skipping` was not correct. Spekk does not skip such a spec. The spec parses correctly, and `spekk status` shows it as `0/0 assertions complete`. That is the correct result for a spec with no assertions.

## A lock shows that a builder holds the assertion now

`spekk validate` made a `locked-by` value necessary on each `in_progress` assertion. A coach cannot obey that rule, because a lock names a builder session and no command makes one.

The coach prompt told the coach to change an edited assertion to `in_progress`. `validate` then refused the tree. The only method to obey both rules was to invent a lock value. An invented lock looks the same as a lock from a builder that stopped.

The code did not agree with the rule. `spekk next` treats an unlocked `in_progress` assertion as available work. `validate` now accepts it.

The opposite rule stays. A `done`, `failed`, `not_started`, or `draft` assertion with a lock is still a failure.

`validate` now reports an old lock. The previous rule made this impossible. It reported a missing lock, and a lock on a completed assertion, but never a lock that is months old, because that shape was legal.

The coach prompt now writes `not_started` when it edits a `done` or `failed` assertion. This returns the assertion to the queue, which was always the intention, and it needs no lock.

## Installs go to the correct directory

`spekk install` now uses the path to identify a file that spekk manages. It writes the current content, and it keeps the previous content in a `.bak` file. Before this release, spekk examined the content of the file to identify it. Thus a change to the text of a shim left each copy in the field out of date.

`spekk install` reports a symbolic link at a managed path, and a file that it cannot read. It does not write to them. Two tools can own that path, and only you can decide which one.

`spekk install --target <tool> --project` and the `spekk update` report now use the repository root. Before this release, they used the current directory. Thus the command in `repo/src` wrote a second install to `repo/src/.claude`, and it did not update `repo/.claude`. The report had the same fault. It did not mention an out-of-date project file one level higher, and the remedy that it printed made the second install.

## Also in this release

- `spekk list --json` and `spekk next --all` now include the `branch` field on spec objects and assertion objects. The parser always read the field, but the flat view and the hierarchy view removed it. Thus an agent needed a second call for each record to find the branch.
- The coach prompt and the builder prompt now recommend `spekk query` to find an assertion. Before, they recommended `spekk list --json` with `grep`. The JSON has indentation, so `grep` returns one line: a title with no id and no file. `grep` also matches a keyword in a path as easily as a keyword in a title. Neither source holds an assertion body, so both show the same fields, and SQL selects them better. Use `grep -rl` to search the text of a spec.

## Upgrading

**`spekk validate` has three new failure conditions. A repository that passes now can fail after you upgrade.** Each condition causes a loss of work that you cannot see. Thus each one is a failure and not a warning:

- A spec directory that has assertion files but no main spec file. The parser removes the full directory, so each assertion in it is not in the queue.
- A path with the name `assertions` that is not a directory.
- An `assertions/` directory that spekk cannot read.

Run `spekk validate` one time with the new binary before you change a `SPEKK_VERSION` pin or a pre-commit `rev`. [Validation in CI and pre-commit](../ci.md) gives the advice about pins for this reason.

A spec directory with no `assertions/` directory still passes. It is not a fault.

No other change in this release makes a tree fail that passed before. The branch guard became more permissive. Both new reports are warnings, and they do not change the exit code.
