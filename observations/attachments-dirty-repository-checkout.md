---
slug: attachments-dirty-repository-checkout
type: code_spec_misalignment
severity: high
status: resolved
created: 2026-10-22T00:00:00Z
announced: 2026-10-22T19:02:42Z
affected:
  - cmd/sandbox/attachments.go
  - cmd/sandbox/invoke.go
  - cmd/sandbox/repository.go
  - specs/sandbox-attachment-downloads/assertions/safe-downloads-before-model.md
  - specs/agent-repository-preparation/assertions/repository-checkout.md
---

# Attachments in a repository turn make the next turn's checkout dirty

## Issue Description

A chat turn that names a repository runs in its prepared checkout. If that turn also has attachments, the agent saves them inside the checkout under `.attachments/turn-<random>/`. Git sees that directory as untracked. The next repository turn for the same repository ID runs `git status --porcelain`, finds `?? .attachments/`, and stops with "repository has local changes". Every later turn on that repository then fails the same way until someone deletes the directory by hand on the sandbox.

This breaks the `agent-chat-repository-checkout` criterion that "a later turn sees the selected ref at its fetched commit". The agent creates the dirty state itself, so the person never made local changes. No assertion covers a repository turn that also has attachments, and the attachment tests use a plain temporary workspace only.

## Evidence

- `cmd/sandbox/invoke.go:19-30`: for a repository turn, `workspace` becomes the directory that `prepareRepository` returns (`<workspace>/repositories/<id>`).
- `cmd/sandbox/invoke.go:71-73` (added in f75988d): `downloadConfig.Workspace = workspace`, so attachments download into the repository checkout.
- `cmd/sandbox/attachments.go:83`: `root := filepath.Join(cfg.Workspace, ".attachments")`. Successful turn directories are never removed, and nothing adds them to `.git/info/exclude`.
- `cmd/sandbox/repository.go` `verifyRepository`: `git status --porcelain` with any output returns "repository has local changes; save them before selecting another ref".
- Checked in a scratch repository: after an untracked `.attachments/turn-abc/1-a.txt` is created, `git status --porcelain` prints `?? .attachments/`.
- `specs/sandbox-attachment-downloads/assertions/safe-downloads-before-model.md` requires the download to go into "its actual workspace", and `specs/agent-repository-preparation/assertions/repository-checkout.md` requires a dirty checkout to fail the turn. Each rule is right on its own, but together they make a repository unusable after its first turn with an attachment.

## Impact

High. One attachment in a repository chat blocks every later turn on that repository, and the error tells the person to save local changes they never made. If the repository is pushed, the attachments could also be committed by a `git add -A`.

## Recommendation

Re-queue `safe-downloads-before-model` (remedy commit flips it to `failed`) and add a criterion that attachments in a repository turn must not make the checkout dirty for the next turn. Options for the builder: write repository-turn attachments outside the checkout (for example under the configured workspace), or exclude `.attachments/` through `.git/info/exclude` when preparing the repository. Add a test where one repository turn has an attachment and a second turn on the same repository succeeds.
