---
id: sandbox-store-keeps-unknown-fields
created: 2026-10-17T00:00:00Z
priority: 2
---

# `sandboxes.json` keeps fields that the CLI does not know

## Problem

`LoadSandboxes` decodes `sandboxes.json` into `SandboxMeta`, and a save writes only the fields of that struct. When a newer CLI wrote a field that an older CLI does not know, the next save by the older CLI removes the field. A destroy of one sandbox also rewrites the records of all the other sandboxes, so it removes their unknown fields too (issue #269).

## Solution

Each record keeps the JSON fields that `SandboxMeta` does not know, and writes them back on save, after the known fields. The known fields keep their order, so a file with no unknown fields is written as before.

## What this deliberately does not do

- **No version field and no migration.** The file stays the same format.
- **No merge with the file on disk at save time.** A record keeps what it read. The read-modify-write in `SaveSandbox` and `RemoveSandbox` is enough.
