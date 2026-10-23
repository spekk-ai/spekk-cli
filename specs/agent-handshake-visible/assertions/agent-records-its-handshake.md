---
id: agent-records-its-handshake
parent: agent-handshake-visible
created: 2026-10-23T06:00:00Z
priority: 1
status: done
---

# The agent client records its last handshake and its last dial failure

**Tests:** `cmd/sandbox/connection_record_test.go`

## Success criteria

- After each dial, the agent client writes `/var/log/spekk/agent-connection.json`. Every provision path and `deployAgent` already make `/var/log/spekk` and give it to the `agent` user.
- The record is one JSON object with these fields:
  - `last_connected_at`: the UTC time (RFC 3339) of the last dial that completed the WebSocket handshake.
  - `last_handshake_failure_at`: the UTC time (RFC 3339) of the last dial that failed.
  - `last_handshake_failure_status`: the HTTP status code of that failure, for example 403. It is absent when the dial got no HTTP response, for example when the host did not answer.
  - `last_handshake_failure_error`: the error text of that failure, at most 200 characters. The token is never in it: the token is only in the `Authorization` header, and a dial error does not contain the headers.
- A success keeps the last failure, and a failure keeps the last success, so the record shows both.
- At start, the agent reads the record that is on the disk, so a restart does not lose the last success.
- The write replaces the file in one step (a temporary file and a rename), so a reader never sees a part of the record.
- A write that fails does not stop the agent. The agent logs the first write failure of the process with its cause, and not each later one, so that a dial loop does not fill the log.

## Test

One test with the real `AgentClient.Run` against a test server that refuses the first dial with HTTP 403 and accepts the next one. Assert that the record shows the 403 failure, and then the success, with the failure still in it.
