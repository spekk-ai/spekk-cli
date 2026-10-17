---
id: create-from-image
parent: sandbox-template-images
created: 2026-10-17T00:00:00Z
priority: 1
status: not_started
depends-on: image-build
branch: feature/sandbox-template-images
---

# `spekk sandbox create --image` Starts From the Image, or Falls Back and Says So

## Success Criteria

- `spekk sandbox create --image <name>` finds the image: first the record in `images.json`, then, when there is no record, an image with that name in the provider's own list.
- When the image is for this provider (and, on DigitalOcean, has a copy in the region of the create, default `nyc1`), the provider creates the machine from it, with a short first-boot cloud-init: the `agent` user with the sandbox key, and the `/opt/spekk/.provisioned` marker as its last step. Cloud-init makes new SSH host keys, and systemd makes a new `/etc/machine-id`, because the image has neither. No package is installed.
- The `/opt/spekk/.provisioned` check, credential injection, and the agent deploy do not change. The deploy still compares Claude Code with the minimum and upgrades an older one, and the agent binary is always deployed from the release, never taken from the image.
- **Fallback:** when there is no image for the provider or the region, create prints one line, `No image "<name>" for <provider> in <region>; creating from the stock image with the full cloud-init.`, and creates the machine as it does with no `--image`.
- `--image` with an existing machine (`--provider none`, `--ip`) is refused with a clear error before anything is created.
- The sandbox record has `image: <name>` when the machine was created from the image. A fallback create has no `image` field.
- When the image is older than 14 days, create prints a warning with its age and the command that rebuilds it. The create continues.

## Verification

- A test of the resolution: a record for the provider and region, a record in another region (the fallback line), and no record at all (the fallback line).
- A test that two machines from one image get different SSH host keys and machine IDs is not possible without a cloud. The walkthrough on the local fake cloud, and the live acceptance script, check it.
