---
id: builder-start-interrupt
parent: builder-cli-flags
created: 2026-11-20T08:15:00Z
priority: 1
status: done
branch: fix/loop-builder-start-race
---

# Interrupt a builder during child startup

While a builder child starts, interrupt handling waits for the start result. If Ctrl+C arrives after the child starts and before the holder records its process, the builder forwards the signal and waits for the child to exit. The holder returns to an idle state after the child exits or fails to start.

**Tests:** internal/agent/builder_start_test.go
