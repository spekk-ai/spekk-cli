---
id: status-shows-kind-fields
parent: sandbox-kind
created: 2026-10-16T00:20:00Z
priority: 2
status: done
depends-on: one-off-create-labels-and-marker
branch: sandbox-kind
---

# `status` Shows the Kind Fields

`spekk sandbox status <name>` is the only command that shows one sandbox in full. Without the kind fields, the `purpose` of a sandbox is visible only in `sandboxes.json` and on the machine.

## Success Criteria

- `spekk sandbox status <name>` prints `Kind`, `Purpose`, `Expires` and `Created by`, from the local record.
- `Kind` is `persistent` for a record with no kind. An empty field prints `-`.
- An expiry in the past shows `EXPIRED` before the time, the same as in `list`.

## Verification

- A walkthrough runs `create --one-off --purpose ... --provider none --ip ...` and then `status`, and compares each value with `sandboxes.json` and the marker on the machine.
