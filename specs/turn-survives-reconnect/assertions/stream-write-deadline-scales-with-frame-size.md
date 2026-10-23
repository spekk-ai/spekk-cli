---
id: stream-write-deadline-scales-with-frame-size
parent: turn-survives-reconnect
created: 2026-10-23T05:00:00Z
priority: 2
status: done
---

# A stream write deadline scales with the size of the frame

**Tests:** `cmd/sandbox/reconnect_test.go`

A stream frame carries one line of the child's stdout, and the scanner accepts a line of up to 10 MB. The stream write deadline was a flat 5 seconds, and an expired write deadline closes the connection. On an uplink slower than about 2 MB/s, a large tool result closed a healthy connection in the middle of a turn, and the next large line did it again (issue #212).

## Success criteria

- The deadline of a stream write is `streamSendTimeout` (5 seconds) plus the time to send the encoded frame at a floor rate of 256 KiB/s. A small frame keeps the 5-second bound. An encoded frame of 10 MB, the scanner's line limit, gets a deadline of less than one minute.
- The bound stays: a connection whose peer stops reading still releases the stdout reader within that deadline.
- The bytes on the wire do not change.

**Note:** 256 KiB/s is a floor for a slow link, not a measured rate. A link slower than the floor still closes on a very large frame, as before.

## Test

One table test of the deadline function: a small frame gets `streamSendTimeout`, and a frame of the maximum line size gets more than the time that the frame needs at 2 MB/s and less than one minute.
