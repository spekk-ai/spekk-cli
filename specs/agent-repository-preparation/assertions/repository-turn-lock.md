---
id: agent-chat-repository-turn-lock
parent: agent-repository-preparation
created: 2026-10-19T21:00:00Z
priority: 1
status: done
branch: feature/repository-chat-preparation
depends-on: agent-chat-repository-checkout
---

# A repository checkout belongs to one turn at a time

**Tests:** cmd/sandbox/repository_test.go

## Success Criteria

- Two concurrent repository turns for the same repository ID cannot change the checkout while either Claude turn uses it. A later turn can prepare a different ref after the first turn ends.
- Repository turns for different repository IDs can run at the same time.
- A chat turn without a repository still uses the configured workspace.
