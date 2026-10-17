---
id: machine-mode-image
parent: sandbox-template-images
created: 2026-10-17T00:00:00Z
priority: 2
status: not_started
depends-on: create-from-image
branch: feature/sandbox-template-images
---

# The Machine-Mode Launch JSON Takes an Optional `image`

## Success Criteria

- The stdin object of `spekk sandbox create --machine` has an optional field `image`, a string. Without it, or with an empty value, nothing changes.
- With it, machine mode finds the DigitalOcean image by that name in the account's private images (the worker has no `images.json`), and creates the droplet from it, as `create --image` does. With no such image in the region, it writes the fallback line to stderr and creates the droplet with the full cloud-init.
- The stage lines do not change: `creating`, `provisioning`, `registering`, `online`, or `failed`. The record has `image` when the droplet came from the image.
- The contract in `specs/sandbox-machine-mode/sandbox-machine-mode.md` shows the field.
