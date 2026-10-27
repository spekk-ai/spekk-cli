---
id: existing-machine-gets-fields-not-labels
parent: sandbox-kind
created: 2026-10-15T23:40:00Z
priority: 1
status: done
depends-on: one-off-create-labels-and-marker
branch: sandbox-kind
---

# A Machine You Already Have Gets the Fields and the Marker, but No Labels

`create --ip` registers a machine that no cloud owns. It has no labels to set, and that is not an error.

**Tests:** internal/sandbox/kind_test.go

## Success Criteria

- `create --ip <address> --ssh-key <path> --one-off --purpose x` records `kind`, `purpose`, `expiresAt` and `createdBy`.
- With a nil provider, `Create` does not try to label and prints no labeling warning.
- The credential injection for this machine writes `/opt/spekk/sandbox.json` the same as for a cloud machine.

## Verification

- A test runs `Create(nil, ...)` with `--one-off` options through to the equip step, and finds the four fields in the record, the marker in the inject script, and no labeling output.
