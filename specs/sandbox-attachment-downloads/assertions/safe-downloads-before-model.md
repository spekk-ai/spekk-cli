---
id: safe-downloads-before-model
parent: sandbox-attachment-downloads
created: 2026-10-21T20:09:18Z
branch: observer/attachments-dirty-repository-checkout
priority: 1
status: done
---

# Required attachments are safe before the model starts

## Success Criteria

- Unique text and image bytes download through the real authenticated HTTP route and remain intact in different private paths.
- Bad status, redirects, cancellation, oversized or incomplete bodies, invalid IDs, unsafe names, and an excessive file count fail the batch. No partial files remain.
- The local harness receives the correct text and image bytes through saved paths for each message. Result frames keep the initiating agent session ID, and earlier message paths do not reach later prompts.
- The worker downloads into the configured agent workspace before it starts the model. For a repository turn, the saved paths are outside the prepared checkout, and the model can read them from the prompt. A later turn can prepare the same repository and selected ref while the checkout stays clean. A user change in the checkout still stops repository preparation.
- A download failure sends a turn error and returns before model startup.
- The existing attachment message fields and download route stay compatible. A real model read of the files remains an acceptance step for the control host.

**Tests:** cmd/sandbox/attachments_test.go
