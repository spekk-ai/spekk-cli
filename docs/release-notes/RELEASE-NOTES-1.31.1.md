# Spekk CLI 1.31.1 - A Failed Turn Says Why

When `claude` exited with an error on a sandbox, the agent reported only `claude exited: exit status 1`. That text was the same for a model API outage and for every other failure. To find the cause, you had to log in to the sandbox and read the Claude Code session transcript.

## The error frame carries the API error

When the model API fails for longer than Claude Code retries, `claude` writes the cause to its final `result` event, for example `API Error: 503 Bedrock is unable to process your request.` The agent now adds that text to the `error` field of the error frame that it sends to the control host, and to the `Claude failed` line in its log on the sandbox:

```
claude exited: exit status 1: API Error: 503 Bedrock is unable to process your request.
```

The added text is capped at 500 characters. A failure with no error result still reports the exit status alone. The frame keeps its other fields, and the protocol version does not change, so a control host needs no change to log the cause.

## Upgrading

Run `spekk sandbox deploy` for each sandbox to install the new agent.
