---
id: tests-ignore-developer-git-config
parent: tests-pass-on-any-machine
created: 2026-11-19T15:58:31Z
priority: 2
status: done
branch: feature/tests-pass-on-any-machine
---

# Tests That Run Git Ignore the Developer's Global and System Git Config

## Success Criteria

- `go test ./...` gives the same pass/fail result when the developer's global git config sets `user.name`, `user.email`, `init.defaultBranch`, `commit.gpgsign = true`, or a `core.hooksPath`, as when it sets none of them.
- `TestResolvesIdentAgreesWithWhatCommitTreeAccepts` and `TestAnnounceMarksTheFindingWhenTheRepoHasNoIdentity` pass on a machine whose global config sets `user.name` and `user.email`. A repository the test configures with no identity has no identity.
- A test helper builds the environment for git subprocesses in tests. Every test that runs git uses it, so no test reads the developer's `~/.gitconfig` or the system config.
- A test proves the isolation: it points the global config at a file that sets an identity, and it asserts that a fixture repository with no identity still has none.

**Note:** Setting `HOME` alone is not enough. Git also reads `XDG_CONFIG_HOME/git/config` and the system config. The isolation must cover all three: for example `GIT_CONFIG_GLOBAL` and `GIT_CONFIG_SYSTEM` set to an empty file, or `GIT_CONFIG_NOSYSTEM=1`.

**Note:** Production code that runs git must still read the user's real config. The isolation applies to tests only.

**Tests:** internal/testenv/testenv_test.go (`TestIsolateGitHidesDeveloperIdentity`), internal/testenv/testenv.go (`Main`, `IsolateGit`), `TestMain` in every package whose tests run git, internal/observer/gitwrite_test.go (`TestResolvesIdentAgreesWithWhatCommitTreeAccepts`, `TestAnnounceMarksTheFindingWhenTheRepoHasNoIdentity`)
