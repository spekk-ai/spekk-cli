---
id: unknown-fields-survive-a-save
parent: sandbox-store-keeps-unknown-fields
created: 2026-10-17T00:00:00Z
priority: 2
status: done
---

# An unknown field survives a read-modify-write of `sandboxes.json`

**Tests:** internal/sandbox/store_test.go

## Success Criteria

- A record with an unknown top-level field and an unknown nested object keeps both, with the same values, after `SaveSandbox` of that record or of a different record, and after `RemoveSandbox` of a different record.
- A known field that the CLI changes gets the new value. An unknown field cannot hide or replace a known field.
- A file with no unknown fields is written byte for byte as before.
- The test fails on `main`.
- Live check: with a fresh `HOME`, put a record with unknown fields next to a second record, run `spekk sandbox destroy <second> --force`, and see that the first record keeps its unknown fields.
