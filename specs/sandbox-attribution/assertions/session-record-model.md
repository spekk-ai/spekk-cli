---
id: session-record-model
parent: sandbox-attribution
created: 2026-10-20T00:00:00Z
priority: 1
status: done
branch: feat/sandbox-attribution
depends-on: agent-attribution-setup
---

# The trailer model comes from this agent session

## Success Criteria

- A Codex session ID selects its own rollout record. If that record is absent, `assisted-by` exits with a clear error and prints no trailer, even when Codex config names a model.
- If Claude Code has no session ID, `assisted-by` uses a project transcript only when that project has one transcript. If the project has two or more transcripts, it exits with a clear error and prints no trailer.
- A valid Codex rollout supplies the model in the trailer. The installed script and repository script have the same bytes.

**Tests:** `scripts/test_assisted_by.py` covers the missing Codex record and ambiguous Claude transcripts. A real Codex session check records the helper output.
