---
id: image-hygiene-broad-patterns
parent: sandbox-template-images
created: 2026-10-18T00:00:00Z
priority: 1
status: done
depends-on: image-hygiene-gate
branch: feature/sandbox-template-images
---

# The Hygiene Check Uses Broad Patterns for Each Home

**Tests:** internal/sandbox/image_test.go

## Success Criteria

- "Each home" is `/root` and each directory in `/home`. The check is a denylist of patterns, not of single file names. It has no allowlist, because a setup can leave harmless dotfiles.
- The clean step removes, in each home, `.ssh/authorized_keys`, each `.*_history` file, and `.viminfo`. It removes `.claude` and `.claude.json` when there is no `.claude/.credentials.json` in that home.
- The check fails the build when, in any home, it finds:
  - a file in `.ssh` other than `known_hosts`, `config`, or a `*.pub` file (this includes a left `authorized_keys`)
  - `.claude/.credentials.json` or `.claude.json`
  - `.netrc`, `.aws/credentials`, `.config/git/credentials`, `.config/gcloud`, `.git-credentials`, or `.config/gh/hosts.yml`
  - a `.*_history` file or `.viminfo`
- The check fails the build when a git `config` file in a `.git` directory under `/root`, `/home`, or `/opt` has an `http://` or `https://` URL with a user part (`https://user:token@host` or `https://token@host`).

## Verification

- `TestBuildImage` has one row for each of these: an SSH private key, a Claude Code sign-in of root, a credential file (`.netrc`) of a user, a token in a remote URL of a clone in `/opt`, and a `.python_history` that the clean step removes. Each row fails on the code before this assertion.
