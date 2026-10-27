---
id: managed-commands-acceptance-run
parent: app-managed-sandbox-client
created: 2026-10-23T00:00:00Z
priority: 3
status: not_started
depends-on: managed-sandbox-commands
---

# A cold run against a local app proves the command workflow

Owner: coordinator (integration QA). This run does not block the merge of the command workflow.

## Success criteria

- A cold terminal run from a fresh home, against a local app with a fake cloud, covers launch, readiness, text send, saved result, continuation, a lost response, client restart, denied member team launch, denied repository push, offline agent, quota refusal, revoked key, and cleanup. It checks the app records for one launch and one message per request ID.
- It fails a launch after machine creation, leaves `finished` null after failed removal, and checks the warning. It then completes cleanup and checks that the ended stage with `finished` set clears the warning. It drops a cancel or destroy response and checks recovery by launch status without a request-ID lookup or an automatic second POST.
- It repeats a send whose dispatch the app did not confirm, and checks the 409, the saved status, and that no second task reaches the agent.
- The run creates no real cloud resource and makes no model request. The transcript goes in the pull request or the QA record.
