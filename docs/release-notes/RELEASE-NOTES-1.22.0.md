# 1.22.0 — Validation Becomes a Gate

`spekk validate` was available before this release, but nothing made it run. This version puts it in the two places where an incorrect frontmatter field is still easy to correct. It also tells the agents to run it, and it makes the error messages give the effect.

## Why one incorrect line is important

Each spekk command builds the index from the same files. Thus one incorrect frontmatter field stops the parse of the full tree, and not only of its own file. On the default branch, this stops `spekk next`, `spekk observer announce`, and `spekk query`. It stops them for each branch and each user, until a person corrects the line.

This occurred. An incorrect `depends-on` value in an observer remedy commit went to the default branch of a repository. It stopped `spekk validate` for each user of that repository, and two scheduled agent runs failed.

## A pre-commit hook and a CI gate

This repository now supplies a `spekk-validate` pre-commit hook. Thus an incorrect line does not become a commit:

```yaml
repos:
  - repo: https://github.com/spekk-ai/spekk-cli
    rev: v1.22.0
    hooks:
      - id: spekk-validate
```

Use both checks, because one alone is not sufficient. The hook gives fast feedback. `spekk validate` takes less than 10 ms, so the author sees the problem before the commit exists. But the hook is not active in a new clone until a person runs `pre-commit install`, and `git commit --no-verify` does not run it. CI is the check that a user cannot bypass. Refer to [Validation in CI and pre-commit](../ci.md).

The hook uses `language: system`. Thus it runs the spekk binary on `PATH`, and it agrees with the version that the developer and the agents use. For this reason, `rev` pins this hook definition and not the binary. Pin the binary in CI with `SPEKK_VERSION`. Without that pin, a more strict validator in a later version can make the repository fail with no change on its side.

## The agents run it

Each agent path that writes to `specs/` now names `spekk validate`. Two of these paths had incorrect instructions. The coach was told to validate with the parser before its edit, which cannot find a problem that the edit causes. The coordinator skill called an internal function and not the command. The observer remedy path had no validation step, and that path caused the failure above.

## An error message that gives the effect

Before this release, an error named the field and stopped:

```
error: auto-rebuild of index failed: cannot parse specs: Field 'depends-on' must be
kebab-case (lowercase with hyphens) in specs/.../a-two.md
```

An agent read that message and wrote `depends-on: a, b, c`. That value fails the same check. The message did not give the effect, and it did not name the command that shows the full result. The message now does both.

## The branch warning stops on correct specs

The warning for the `branch` field accepted only gitflow names: `main`, `master`, `develop`, `feature/`, `bugfix/`, `hotfix/`, and `release/`. A project that uses conventional-commits prefixes got a warning on each assertion with the field. This repository gave 73 warnings on its own specs.

These prefixes now pass: `feat/`, `fix/`, `chore/`, `docs/`, `refactor/`, `test/`, `perf/`, `build/`, `ci/`, and `style/`. A dot also passes, so `release/1.22.0` is no longer an error. The rules from git stay: no `..`, no final `.`, and no `.lock` suffix.

Two corrections came with this change. The message listed four accepted values, but the pattern accepted seven. Thus spekk accepted `develop`, `master`, and `release/` but did not document them. The message and the pattern now come from one list.

This release also removes the `branch` field from 69 done assertions, from one parent spec, and from three `not_started` assertions. The field has no effect on a done assertion, because `spekk next` removes done assertions before it applies the branch filter. 51 of the 69 named a branch that no longer exists.
