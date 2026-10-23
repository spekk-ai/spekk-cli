---
id: repository-git-failure-shows-cause
parent: agent-repository-preparation
created: 2026-10-23T15:30:00Z
priority: 1
status: done
branch: fix/repository-git-stderr
depends-on: agent-chat-repository-checkout
---

# A failed Git step of repository preparation shows the cause

**Tests:** cmd/sandbox/repository_test.go

Before this change, the agent client discarded the stderr of each Git command of repository preparation. A failed clone gave only "git clone failed: exit status 128", in the turn failure reason and in the agent log. Exit status 128 has many causes: a refused credential, a credential helper that cannot read its settings, a missing repository, or a network failure. The operator could not tell them apart without a new run on the host.

## Success Criteria

- When a Git command of repository preparation fails, its error keeps the last lines of the Git stderr after the exit status. The turn failure reason and the agent log both show it.
- The kept text has no credential: the user name and password part of a URL, a GitHub token, and the value of an `Authorization` header become `<redacted>`.
- The kept text is short: at most the last five non-empty lines, and at most 600 characters. The failure reason stays below the 1000-character limit of the control host.
- The credential helper error names the cause when it cannot read `/etc/spekk/agent.env` (for example "permission denied"). The error has no value from the file.
- A successful Git command writes nothing new to the error or the log.
