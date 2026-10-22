---
id: safe-downloads-before-model
parent: sandbox-attachment-downloads
created: 2026-10-21T20:09:18Z
branch: fix/chat-attachment-downloads
priority: 1
status: failed
---

# Required attachments are safe before the model starts

## Success Criteria

- Unique text and image bytes download through the real authenticated HTTP route and remain intact in different private paths.
- Bad status, redirects, cancellation, oversized or incomplete bodies, invalid IDs, unsafe names, and an excessive file count fail the batch. No partial files remain.
- The local harness receives the correct text and image bytes through saved paths for each message. Result frames keep the initiating agent session ID, and earlier message paths do not reach later prompts.
- The worker downloads into its actual workspace before it starts the model. A download failure sends a turn error and returns before model startup.
- The existing attachment message fields and download route stay compatible. A real model read of the files remains an acceptance step for the control host.

**Tests:** cmd/sandbox/attachments_test.go
