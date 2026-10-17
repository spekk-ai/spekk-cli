---
id: image-docs-and-acceptance
parent: sandbox-template-images
created: 2026-10-17T00:00:00Z
priority: 2
status: not_started
depends-on: create-from-image
branch: feature/sandbox-template-images
---

# The Docs and an Acceptance Script Cover Template Images

## Success Criteria

- `docs/cli-reference.md` documents `image build`, `image list`, `image delete`, `create --image`, the fallback, the 14-day warning, and the hygiene check. The machine-mode section shows the `image` field.
- The docs show, for AWS, how to put an AMI ID in an SSM parameter and pass its name to the CloudFormation template as `UbuntuAmi`. spekk does not call AWS.
- `scripts/image-acceptance.sh` runs the live acceptance on DigitalOcean from a control host: it builds a base image and a project image (with a setup script that the operator gives), creates a sandbox from each and a fresh one, times each create, checks that two sandboxes from one image have different SSH host keys and `/etc/machine-id` values, runs a Playwright smoke test and `just up` in the project sandbox and checks that no image was pulled, and prints a timing table. It destroys each sandbox and image it made, also on failure, and prints what it removed.
