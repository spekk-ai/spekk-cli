---
id: image-snapshot-prefix
parent: sandbox-template-images
created: 2026-10-18T00:00:00Z
priority: 2
status: not_started
depends-on: machine-mode-image
branch: feature/sandbox-template-images
---

# The DigitalOcean List Fallback Trusts Only Snapshots That `image build` Made

**Tests:** internal/sandbox/machine_test.go, internal/sandbox/doapi_test.go

## Success Criteria

- On DigitalOcean, `image build --name X` names the snapshot `spekk-image-X`. The record in `images.json` keeps the name `X`.
- `ListImages` of DigitalOcean returns only private images whose name starts with `spekk-image-`, with the prefix removed from the name. A private snapshot named `X` that the build did not make is not found, so `create --image X` and machine mode fall back to the full cloud-init.
- This is the same guard as the `spekk-image` label on GCP.

## Verification

- The machine-mode test has two private snapshots, `web-app` and `spekk-image-web-app`. The droplet starts from the second.
