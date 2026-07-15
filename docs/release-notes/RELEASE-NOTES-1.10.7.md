# Spekk CLI 1.10.7 — Init Creates the README in Pre-Existing `specs/`

A fast-follow fix for 1.10.6. On a project whose `specs/` directory predates the orientation feature — the directory exists but has no `README.md` — `spekk init` printed "specs/ already exists — you're set." and wrote nothing, so the managed `specs/README.md` was never created.

`spekk init` now handles all four states of an existing `specs/` directory:

- **No `README.md`** → creates the managed README (this fix).
- **Legacy README, no fence** → preserves it, appends one managed block.
- **Well-formed fence** → regenerates in place (byte-identical if unchanged).
- **Corrupt fence** → recovers to one clean region.

## Upgrade

```bash
spekk update
```
