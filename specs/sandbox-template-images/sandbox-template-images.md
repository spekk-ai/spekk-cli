---
id: sandbox-template-images
created: 2026-10-17T00:00:00Z
priority: 1
---

# Sandboxes Start From a Prepared Template Image

## Problem

Each `spekk sandbox create` starts from a stock Ubuntu image and installs everything again in cloud-init: system packages and upgrades, Docker, Node, the GitHub CLI, Claude Code, and the spekk CLI. A create takes about 4 to 5 minutes, and about 3.5 minutes of that is cloud-init. Each long cloud-init run is also a chance for a failure: a package mirror that is slow, or a repository that is down. A project that needs a browser for its checks must also install Chromium and its system dependencies, and pull its Docker images, before it can start the real work.

Issue #266 has the full design.

## Solution

An optional provider capability makes and uses images, the same approach as the labels of `sandbox-kind`.

- `spekk sandbox image build` makes an image: a temporary builder sandbox with the normal cloud-init, an optional project setup script, a hygiene step that must pass, and a provider image. The builder is always destroyed.
- `spekk sandbox create --image <name>` creates the machine from the image. Only the first-boot steps run: user keys, SSH host keys, the `/opt/spekk/.provisioned` marker, then the normal credential injection and agent deploy. With no image for the provider or region, create uses the normal cloud-init and prints one line that says so.
- `spekk sandbox image list` and `spekk sandbox image delete` keep the images in order.
- The CLI stays generic. A project keeps its own setup script in its own repository.

## Decisions

- **The marker is removed from the image.** The first-boot cloud-init writes it again as its last step, so the `/opt/spekk/.provisioned` check means the same thing on both paths: the first boot is complete.
- **Credentials are not removed, they fail the build.** A setup script that left a credential on the builder can also have left a copy that the hygiene step does not know about. A failed build is safer than an image that was cleaned silently.
- **The image record is local** (`images.json` beside `sandboxes.json`). Machine mode has no local record, so it finds the image by name with the provider's own list.
- **AWS: docs only.** The CloudFormation template already takes the AMI through its `UbuntuAmi` parameter, which reads an SSM parameter. The docs show how to put an AMI ID in an SSM parameter. The template does not change.

Rejected: a scheduled rebuild in the CLI (a follow-up, outside of the CLI), an `apt upgrade` at first boot (it removes the time saving), and a generic "image recipe" format (the setup script is a shell script, which is enough).
