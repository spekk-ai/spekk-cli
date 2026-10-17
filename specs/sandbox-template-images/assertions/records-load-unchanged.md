---
id: records-load-unchanged
parent: sandbox-template-images
created: 2026-10-17T00:00:00Z
priority: 1
status: not_started
branch: feature/sandbox-template-images
---

# Records That v1.33.0 Wrote Load and Save Unchanged

## Success Criteria

- The new `image` field of a sandbox record is optional (`omitempty`). A `sandboxes.json` that v1.33.0 wrote loads, and a save writes it back byte for byte.
- No other file that v1.33.0 wrote changes its format.

## Verification

- One test loads a `sandboxes.json` in the v1.33.0 format, saves it, and compares the bytes.
