---
id: turn-failure-reports-its-cause
created: 2026-10-15T16:30:00Z
priority: 2
---

# A failed turn reports why it failed

## Problem

When `claude` exits non-zero, the sandbox agent client sends an error frame whose `error` field is only `claude exited: exit status 1`, and it writes the same text to its log. That text does not tell a model API outage from any other failure.

The cause is available. When the model API fails for longer than Claude Code retries, `claude` writes the cause to the final `result` event on stdout, with `is_error` set, for example `API Error: 503 Bedrock is unable to process your request.` It writes nothing useful to stderr. The client reads that event for a successful turn, and drops it for a failed one.

## Evidence

Scheduled runs failed in production on a model API outage. Each one gave up after 10 retries on 503 errors in the first request. The control host log, the sandbox log, and the alert all showed only `exit status 1`. To find the cause, an operator had to log in to the sandbox and read the Claude Code session transcript.

## Solution

When `claude` exits non-zero and its last `result` event has `is_error` set, add the `result` text to the `error` field of the error frame and to the `Claude failed` log line.

## What this deliberately does not do

- **No protocol change.** The `error` field is a string before and after. The control host already logs it, so the cause reaches the control host log with no control-host change.
- **No retry of the turn.** A turn that failed after it did work, and then runs again, can repeat that work. The retry count of Claude Code is the setting for a longer API outage.
