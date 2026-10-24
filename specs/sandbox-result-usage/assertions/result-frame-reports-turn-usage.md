---
id: result-frame-reports-turn-usage
parent: sandbox-result-usage
created: 2026-10-24T02:14:00Z
priority: 1
status: done
branch: feature/result-usage
---

# The result frame reports valid usage for one turn

## Success criteria

- A successful turn with a valid Claude Code result event sends one optional `usage` object on its result frame. The object has exactly `input_tokens`, `output_tokens`, `cache_read_input_tokens`, `cache_creation_input_tokens`, and `cost_usd`. The four counts are JSON integers from 0 to 10,000,000,000. The cost is a finite JSON number from 0 to 1000.
- A first process reports its `total_cost_usd` as this turn's cost. A resumed process reports the difference between its `total_cost_usd` and the preceding total for that Claude session. A resumed process with no known preceding total, a changed session ID, or a negative difference sends no usage.
- A missing or malformed count or cost sends no usage. A result event with no usage still ends the turn. A later valid result can establish a cost baseline even when its token counts are malformed.
- An error frame never carries usage. The existing `type`, `session_id`, `agent_session_id`, `output`, and `terminal` fields keep their values. The protocol major remains 1.

**Tests:** `cmd/sandbox/invoke_test.go`
