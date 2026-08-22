# 1.22.0 — Validation Becomes a Gate

`spekk validate` existed before this release. What it lacked was anything that made it run. This version puts it in the two places where a malformed field is still cheap to fix, teaches the agents to call it, and makes its failures say what they cost.

## Why one bad line matters

Every spekk command builds the index from the same files, so one malformed frontmatter field fails the parse of the **whole tree**, not only its own file. On the default branch that stops `spekk next`, `spekk observer announce`, and `spekk query` for every branch and every user, until a person fixes the line.

That is not hypothetical. A malformed `depends-on` in an observer remedy commit reached the default branch of a repository under observation, broke `spekk validate` for everyone there, and failed two scheduled agent runs.

## A pre-commit hook, and a CI gate

The repository now publishes a `spekk-validate` pre-commit hook, so a bad line never becomes a commit:

```yaml
repos:
  - repo: https://github.com/spekk-ai/spekk-cli
    rev: v1.22.0
    hooks:
      - id: spekk-validate
```

Two checks, because neither is enough alone. The hook is the fast feedback — `spekk validate` runs in under 10 ms, so the author hears about it before the commit exists. But the hook is inactive in a fresh clone until somebody runs `pre-commit install`, and `git commit --no-verify` walks past it. CI is the check that cannot be skipped. See [Validation in CI and pre-commit](ci.md).

The hook uses `language: system`, so it runs the spekk already on `PATH` and agrees with the version the developer and the agents use. That means `rev` pins this hook definition, not the binary — pin the binary in CI with `SPEKK_VERSION`. Without that pin, a stricter validator in a later version turns the repository red with no change on its side.

## The agents run it

Every agent path that writes to `specs/` now names `spekk validate`. Two of those were not weak wording but wrong instructions: the coach was told to "validate with parser" **before** its edit, which cannot catch what the edit introduces, and the coordinator skill called an internal function rather than the command. The observer's remedy path had no validation step at all, which is the path that caused the outage above.

## An error that names its effect

Before, an error named the field and stopped:

```
error: auto-rebuild of index failed: cannot parse specs: Field 'depends-on' must be
kebab-case (lowercase with hyphens) in specs/.../a-two.md
```

An agent read that message and proposed `depends-on: a, b, c`, which fails the same check. The message gave no effect and never named the command that would have shown the whole picture. It now says what broke and what to run.

## The branch warning stops firing on healthy specs

The `branch` field's warning accepted gitflow names only — `main`, `master`, `develop`, `feature/`, `bugfix/`, `hotfix/`, `release/`. A project using conventional-commits prefixes got a warning on every assertion carrying the field, and this repository gave 73 on its own specs.

`feat/`, `fix/`, `chore/`, `docs/`, `refactor/`, `test/`, `perf/`, `build/`, `ci/`, and `style/` now pass. A dot passes too, so `release/1.22.0` is no longer an error, while git's own rules still hold: no `..`, no trailing `.`, no `.lock` suffix.

Two corrections came with it. The message listed four accepted values while the pattern accepted seven, so `develop`, `master`, and `release/` were accepted and undocumented; both now come from one list. And the `branch` field was removed from 69 done assertions, one parent spec, and three `not_started` ones. On a done assertion the field has no effect, because `spekk next` removes done assertions before it applies the branch filter, and 51 of the 69 named a branch that no longer existed.
