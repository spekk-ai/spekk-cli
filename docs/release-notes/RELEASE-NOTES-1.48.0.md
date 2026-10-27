# Spekk CLI 1.48.0 - Turn Usage in Sandbox Result Frames

This release adds one optional field to the sandbox protocol. The `result` frame that ends a turn now carries a `usage` object, so spekk-app can show the model use of each sandbox turn. The protocol version changes from `1.5` to `1.6`. There are no new commands or flags.

## The result frame reports the model use of the turn

When Claude Code exits with code 0, the agent client on the sandbox adds `usage` to the `result` frame. This example uses the values from a recorded turn in the tests:

```json
{
  "type": "result",
  "session_id": "<claude session id>",
  "agent_session_id": "<agent session id>",
  "output": "<reply text>",
  "terminal": true,
  "usage": {
    "input_tokens": 8,
    "output_tokens": 318,
    "cache_read_input_tokens": 45343,
    "cache_creation_input_tokens": 6610,
    "cost_usd": 0.0341703
  }
}
```

- **Token counts.** The four counts come from the `modelUsage` object on the last Claude Code result event of the process. The client adds the counts of all models in that object, so the counts include the model calls of subagents, background tasks, and compaction. `modelUsage` gives totals for the full Claude session. Thus, for a resumed session, the client sends the difference from the preceding result of that session. The value is the model use of this turn only.
- **Cost.** `cost_usd` comes from `total_cost_usd` on the same result event, with the same difference for a resumed session. It is the estimate of Claude Code at API list prices. It is not a bill.

## When the client sends no usage

The client sends the full object or no object. It sends no `usage` when:

- A value is missing or invalid. For example, `modelUsage` is missing or empty, a model entry has a missing or bad count, or `total_cost_usd` is missing, negative, or not a finite number. Each count must be a JSON integer from 0 to 10,000,000,000. The cost of the turn must be from 0 to 1000.
- A resumed turn has no known preceding total for its Claude session, or the session ID changed. The client keeps the totals in memory only, so the first resumed turn after the agent client restarts has no usage.
- The difference from the preceding total is negative, for the cost or for a token count.
- A result had a valid cost but bad token counts. The next resumed turn then has no token baseline and sends no usage. The turn after it reports usage again.
- The turn failed. An `error` frame never carries `usage`.

A frame without `usage` still ends the turn, and the other `result` fields keep their values.

## Compatibility

The field is additive, and the protocol major stays `1`. An agent at `1.6` and a control host at an older `1.x` version still connect. An older spekk-app server ignores `usage`, because it reads only the `result` fields that it knows. A spekk-app server that reads `usage` accepts a frame without it, which is what an agent older than 1.48.0 sends. Thus you can update the agent and the server in either order. A sandbox reports usage only after its agent client runs 1.48.0.
