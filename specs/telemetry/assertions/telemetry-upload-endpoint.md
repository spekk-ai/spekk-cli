---
id: telemetry-upload-endpoint
parent: telemetry
created: 2026-04-08T17:00:00Z
priority: 2
status: not_started
depends-on: telemetry-redaction
branch: feature/telemetry
---

# Telemetry Upload Endpoint

## What Must Be True

A telemetry HTTP client uploads events from the local queue to the configured endpoint. Uploads are batched, retried on transient failures, and respect the user's configured endpoint URL. The client never uploads when telemetry is disabled.

## Success Criteria

- ✅ New package `internal/telemetry/upload/`
- ✅ `Client` struct with `Endpoint string`, `HTTPClient *http.Client`, `InstallID string`
- ✅ `Client.Upload(events []QueuedEvent) (UploadResult, error)` sends events as a single HTTP POST
- ✅ Request format:
  - Method: `POST`
  - Content-Type: `application/json`
  - User-Agent: `spekk-cli/{version}`
  - Header `X-Spekk-Install-ID: {install_id}`
  - Body: `{"events": [<event1>, <event2>, ...]}`
- ✅ Response parsing:
  - `200 OK` → all events accepted, return IDs to delete from queue
  - `207 Multi-Status` with per-event accept/reject → delete accepted, keep rejected
  - `4xx` (except 429) → permanent failure, drop events from queue with warning
  - `429` / `5xx` / network error → transient failure, keep in queue, backoff
- ✅ Batch size limit: 100 events per request (configurable constant, not user-configurable for MVP)
- ✅ Request body size limit: 5 MB (abort batch if exceeded, reduce batch size)
- ✅ Request timeout: 30 seconds
- ✅ Transient failures retried with exponential backoff: 1s, 2s, 4s, 8s, then give up for this invocation
- ✅ `Client.UploadAll(queue *Queue)` processes the full queue in batches until empty or transient failure
- ✅ Pre-upload guard: `if !telemetry.IsEnabled() { return ErrDisabled }` — confirmed by unit test
- ✅ Unit tests with `httptest.Server`: successful upload, 4xx drops events, 5xx keeps events, 429 retries, network timeout, partial 207 response
- ✅ Integration test against a local fake collector (stood up via `httptest`)

## Endpoint Resolution

1. If `endpoint` is set in `telemetry.yaml`, use it
2. Else default to `https://telemetry.spekk.ai/v1/events`
3. No environment variable override for MVP (keeps the story simple)

## Out of Scope

- Compression (gzip) — future optimization
- Authentication beyond install ID header
- mTLS / client certs
- Streaming upload of a single large event

## Notes

The endpoint URL is explicitly configurable so that (a) users can run their own telemetry collector, (b) enterprise customers can route to internal infrastructure, and (c) we can point tests at a local fake without environment tricks.
