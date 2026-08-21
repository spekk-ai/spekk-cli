---
id: project-scope-is-the-repository
parent: agent-install
created: 2026-08-21T00:00:00Z
priority: 2
status: done
depends-on: install-command
---

# A Project Install Is Scoped to the Repository, Not the Working Directory

## Description

`--project` writes the agent shim and the skills into the project. The project is the repository, not the directory the user happens to stand in. `spekk init` already reads it that way: it creates `specs/` at the repository root from any directory inside the repository.

The install did not. It joined the working directory, so the same command gave a different answer from a subdirectory: `spekk install --target claude-code --project` run in `repo/src` wrote a second install to `repo/src/.claude/` instead of updating `repo/.claude/`. The stale report inherited the same rule, so a stale project file one level up was silent, and no warning reads as "the project install is current". The two faults compound: the report sends the user to a command that makes the problem worse.

A directory that is not in a repository has no root to find. The working directory stands in for the project there, which keeps the behavior for a user who installs into a plain directory.

## Success Criteria

- `repoRoot` in `cmd/spekk` returns the repository root for the working directory, and the working directory itself when there is no repository. It reads the root with `git rev-parse --show-toplevel`, the same call `spekk init` uses, so the two commands cannot disagree about where the project is.
- `findSpecsDir` is built on `repoRoot`, so one function holds the rule.
- `scopeForInstall` returns the home directory and the project directory together. The install and the report both take their scope from it, so neither can be given a project the other does not use.
- `spekk install --target <tool> --project` run from any directory inside a repository writes the same files as the same command run at the root, and writes nothing under the subdirectory.
- `spekk update` reports a stale project file from any directory inside the repository that holds it.
- Tests cover: the root from a subdirectory three levels down; a directory outside a repository; `specs/` at the root from a subdirectory; the scope a project install is given; the files an install writes from a subdirectory, and the absence of a second copy under it; and the stale report naming a project file edited at the root while the working directory is a subdirectory.

**Tests:** `cmd/spekk/repo_root_test.go` — `TestRepoRoot_FromASubdirectory`, `TestRepoRoot_OutsideARepository`, `TestFindSpecsDir_FromASubdirectory`, `TestScopeForInstall_ProjectIsTheRepository`, `TestTargetInstallOptions_ScopeIsTheRepository`, `TestProjectInstall_FromASubdirectoryWritesToTheRepositoryRoot`, `TestReportStale_SeesTheProjectInstallFromASubdirectory`.
