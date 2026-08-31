---
icon: lucide/package-check
---

# Cutting a release

Pushing a `v*` tag is the whole release: [`publish.yml`](https://github.com/spekk-ai/spekk-cli/blob/main/.github/workflows/publish.yml) runs the tests, builds every target, and creates the GitHub release. Everything else on this page is what has to be true before that tag exists, and what has to happen after it.

The steps after the tag are the ones that get forgotten, because the release looks finished once the binaries are published. It is not: a pin left behind means the validator guarding a repository's specs is older than the one its team runs, and a fleet left behind runs code the release notes describe as fixed.

## Before the tag

1. **`main` is green.** `go build ./...`, `go vet ./...`, `gofmt -l .` clean, `go test ./... -count=1`, and `spekk validate`. CI runs all of these, so a green `main` is the check.
2. **Write the release notes.** Add `docs/release-notes/RELEASE-NOTES-<version>.md`, link it from `docs/release-notes/index.md`, and add it to the `Release Notes` section of the nav in `zensical.toml`. The nav entry carries the title, so it has to say what the release is for rather than repeat the number.

## The tag

```bash
git tag v1.26.0
git push origin v1.26.0
```

`v*` publishes a stable release; `exp-*` publishes a prerelease, and its version string has the `exp-` prefix stripped. A tag is the only trigger that stamps a real version — a `workflow_dispatch` run falls back to `git describe`.

## After the tag

3. **Raise the pins.** Both examples in [`ci.md`](ci.md) — the pre-commit hook `rev` and `SPEKK_VERSION` — and the same two pins in every downstream project that validates specs in CI. They deliberately do not float, so nothing raises them but a person. Confirm `spekk validate` exits 0 at the new version against that project's tree **before** merging its bump, not after.
4. **Update deployed sandboxes.** A sandbox runs two things from a release: the `spekk` CLI and the agent binary. Both are worth updating together, so a sandbox is not half a release behind.

## Known sharp edges

**A deployed agent may report `version: dev`.** Before the fix in #214 the release build did not stamp the agent, so every agent published up to that point reports `dev` whatever release it came from. Auditing those means comparing a SHA-256 sum against the release asset rather than asking the binary.

**`spekk sandbox deploy` cannot replace a running agent.** `scp` cannot write an executing binary, and the deploy fails with `dest open ... Failure`. Until #214 is closed, replacing it means copying beside the old one, stopping the service, moving it into place, and starting again.

**Release notes are generated as well as written.** `generate_release_notes: true` means GitHub appends its own commit summary to the body. The hand-written notes are the part that says what the release is *for*; the generated part is the changelog.
