---
icon: lucide/package-check
---

# Cutting a release

A push of a `v*` tag is the release: [`publish.yml`](https://github.com/spekk-ai/spekk-cli/blob/main/.github/workflows/publish.yml) runs the tests, builds every target, and creates the GitHub release. This page lists what must be true before the tag exists, and what must happen after it.

The steps after the tag are the ones that get forgotten, because the release looks finished when the binaries are published. It is not finished. A pin left behind means that the validator that guards a repository's specs is older than the one its team runs. A fleet left behind runs code that the release notes describe as fixed.

## Before the tag

1. **`main` is green.** `go build ./...`, `go vet ./...`, `gofmt -l .` clean, `go test ./... -count=1`, and `spekk validate`. CI runs all of these, so a green `main` is the check.
2. **Write the release notes.** Add `docs/release-notes/RELEASE-NOTES-<version>.md`, link it from `docs/release-notes/index.md`, and add it to the `Release Notes` section of the nav in `zensical.toml`. The nav entry carries the title, so it must say what the release is for, not repeat the number. `scripts/check-docs-nav.sh` fails when a release note is not in the nav, and CI runs it.
3. **Review the docs.** A flag, a command, or a default that the release changed must be in `docs/cli-reference.md` and `docs/configuration.md` as well as in the release notes.

## The tag

```bash
git tag v1.28.0
git push origin v1.28.0
```

`v*` publishes a stable release. `exp-*` publishes a prerelease, and its version string has the `exp-` prefix removed. A tag is the only trigger that stamps a version. A `workflow_dispatch` run falls back to `git describe`.

Cut an experimental build with an `exp-*` tag on the feature branch that holds the work. Every release the workflow creates lists `main` as its target, whatever branch the tag is on. The tag's commit is the truth. When you delete an `exp-*` tag, its release stays behind as a draft, so delete the release too.

## After the tag

1. **Raise the pins.** Both examples in [`ci.md`](ci.md), the pre-commit hook `rev` and `SPEKK_VERSION`, and the same two pins in every downstream project that validates specs in CI. They do not float on purpose, so nothing raises them but a person. Confirm that `spekk validate` exits 0 at the new version against that project's tree before you merge its change, not after.
2. **Update the deployed sandboxes.** A sandbox runs two things from a release: the `spekk` CLI and the agent binary. Update both together, so a sandbox is not half a release behind. `spekk sandbox deploy <name>` installs the agent binary.

## Known sharp edges

**A deployed agent may report `version: dev`.** Before 1.26.0 the release build did not stamp the agent binary, so every agent deployed from an earlier release reports `dev`, whatever release it came from. To audit those, compare a SHA-256 sum against the release asset instead of asking the binary. Issue [#214](https://github.com/spekk-ai/spekk-cli/issues/214) tracks the rest of this problem.

**`spekk sandbox deploy` cannot replace a running agent on a root login.** `scp` cannot write a binary that is executing, and the deploy fails with `dest open ... Failure`. Until #214 is closed, replace it by hand: copy the new binary beside the old one, stop the service, move the new binary into place, and start the service. A sandbox with `--ssh-user` does not have this problem, because the deploy stages the binary in the login user's home and `sudo mv` renames it into place.

**Release notes are generated as well as written.** `generate_release_notes: true` means GitHub appends its own commit summary to the body. The hand-written notes say what the release is for. The generated part is the changelog.
