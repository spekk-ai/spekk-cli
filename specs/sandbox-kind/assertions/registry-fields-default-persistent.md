---
id: registry-fields-default-persistent
parent: sandbox-kind
created: 2026-10-15T23:40:00Z
priority: 1
status: done
branch: sandbox-kind
---

# An Old Registry Loads Unchanged and Lists as Persistent

A `sandboxes.json` that v1.32.0 wrote must load, list and save without change. Its records have no `kind`, so they are persistent sandboxes.

**Tests:** internal/sandbox/kind_test.go

## Success Criteria

- `SandboxMeta` in `internal/sandbox/store.go` has the fields `Kind` (`kind`), `Purpose` (`purpose`), `ExpiresAt` (`expiresAt`) and `CreatedBy` (`createdBy`). Each JSON tag has `omitempty`, so a record without them writes no new keys.
- The kind values are the constants `KindOneOff = "one-off"` and `KindPersistent = "persistent"`. A method on `SandboxMeta` returns `persistent` for an empty `Kind`.
- A fixture file in `internal/sandbox/testdata/` holds a `sandboxes.json` in the v1.32.0 format, with a DigitalOcean record, a GCP record and a `provider: none` record. A test loads it, lists it, and finds each record shown as `persistent` and none shown as expired.
- A test saves a new record next to the fixture records and finds the fixture records unchanged on disk: no new keys and no lost keys.

## Verification

- `go test ./internal/sandbox -run Legacy -count=1` passes.
