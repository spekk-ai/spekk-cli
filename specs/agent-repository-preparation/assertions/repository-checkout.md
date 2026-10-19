---
id: agent-chat-repository-checkout
parent: agent-repository-preparation
created: 2026-10-19T20:40:00Z
priority: 1
status: done
branch: feature/repository-chat-preparation
depends-on: agent-chat-repository-access
---

# The selected ref is ready before Claude starts

**Tests:** cmd/sandbox/repository_test.go

## Success Criteria

- A repository turn clones an absent repository into a directory for its repository ID under the agent workspace. It fetches an existing repository and checks out the selected branch. A later turn sees the selected ref at its fetched commit.
- A bad ref, dirty checkout, failed clone, failed fetch, or failed checkout ends the turn with a clear error before Claude starts. A failed operation does not replace a different repository or discard local work.
- A selected checkout is its own Git worktree. Git must not use a parent worktree from a directory at the selected path.
- A ref change does not replace an ignored local file that the selected ref tracks.
- Claude starts in the prepared repository directory. A message with no repository still starts in the configured workspace.
- A real local repository and a clean workspace prove the complete preparation path, including a second turn that fetches a new commit.
