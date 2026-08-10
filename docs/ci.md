---
icon: lucide/shield-check
---

# Validation in CI and pre-commit

`spekk validate` checks every spec and assertion against a fixed set of invariants and exits non-zero when one is violated. Run it in two places: a pre-commit hook, so a bad line never becomes a commit, and CI, so nothing that skipped the hook reaches the default branch.

## Why both

One malformed frontmatter field fails the parse of the **whole tree**, not just its own file. Once that lands on the default branch, every command that rebuilds the index stops working — `spekk next`, `spekk observer announce`, `spekk query` — for every branch and every user, until a person fixes the line. Scheduled agents keep running and keep reporting nothing.

The hook is the fast feedback: `spekk validate` itself takes under 10 ms, so the author hears about it before the commit exists. The hook is not sufficient on its own, because it is inactive in a fresh clone until somebody runs `pre-commit install`, and `git commit --no-verify` bypasses it. CI is the check that cannot be skipped.

## Pre-commit hook

Add this to `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/spekk-ai/spekk-cli
    rev: v1.22.0
    hooks:
      - id: spekk-validate
```

Then run `pre-commit install` once per clone.

`rev` must name a release that contains `.pre-commit-hooks.yaml` — v1.22.0 or later. An earlier tag fails with `InvalidManifestError`.

The hook runs the `spekk` already on your `PATH`, so it agrees with the version you and your agents use day to day. `rev` pins the hook definition, not the binary.

It runs only when a staged file is under `specs/`, so a repository that has not adopted spekk never invokes it, and a commit that touches no spec pays nothing.

## GitHub Actions

Save this as `.github/workflows/specs.yml`:

```yaml
name: Specs

on:
  pull_request:
  push:
    branches: [main]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6

      # Stay green in a project that has not adopted spekk yet, without
      # downloading a binary it will not use.
      - name: Look for specs
        id: check
        run: |
          if [ -d specs ]; then
            echo "found=true" >> "$GITHUB_OUTPUT"
          else
            echo "found=false" >> "$GITHUB_OUTPUT"
          fi

      - name: Install spekk
        if: steps.check.outputs.found == 'true'
        env:
          SPEKK_VERSION: v1.22.0
        run: curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh

      - name: Validate specs
        if: steps.check.outputs.found == 'true'
        run: ~/.local/bin/spekk validate
```

Pin `SPEKK_VERSION` to a release tag. Without it the install takes the latest release, and a stricter validator in a future version would turn the repository red with no change on its side. Raise the pin when you choose to.

## What passes and what fails

| Result | Exit code | Effect |
|---|---|---|
| Valid tree | 0 | Check passes; a one-line summary is printed |
| No `specs/` directory | 0 | Check passes — `validate: 0 specs, 0 assertions OK` |
| Warnings only (for example a non-standard `branch` value) | 0 | Check passes; warnings print to stderr |
| Any invariant violated | 1 | Check fails, naming the file and the offending value |

Warnings do not fail the check. A repository adopting spekk usually has some, and failing on them on the first day teaches people to ignore the result.

The invariants are listed under [`spekk validate`](cli-reference.md#spekk-validate).
