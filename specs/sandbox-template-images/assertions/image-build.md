---
id: image-build
parent: sandbox-template-images
created: 2026-10-17T00:00:00Z
priority: 1
status: done
branch: feature/sandbox-template-images
---

# `spekk sandbox image build` Makes a Provider Image and Always Destroys Its Builder

**Tests:** internal/sandbox/image_test.go, internal/sandbox/doapi_test.go, internal/sandbox/gcp_provider_test.go

## Success Criteria

- `spekk sandbox image build --name <image> [--setup <script>] [--size <slug>] [--provider digitalocean|gcp] [--regions <r1,r2>] [--project <id>]` exists. The image name matches `^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$`, which both providers accept. A name that `images.json` already has is refused before anything is created.
- Image support is an optional interface, `Imager`, with `CreateImage`, `DeleteImage`, and `ListImages`. `CreateOptions.Image` is the create-from-image option. DigitalOcean and GCP implement it. A provider without it, and `--provider none`, are refused with a clear error.
- The build creates a builder sandbox named `image-<image>` with the normal cloud-init, records it in `sandboxes.json` (status `building`, kind one-off), and waits for the `/opt/spekk/.provisioned` marker. It registers no agent, injects no credential, and deploys no agent.
- With `--setup`, the build sends the script to the builder on the stdin of `ssh` and runs it as root. Its output goes to the terminal. A script that exits with an error fails the build.
- The build runs the hygiene step (`image-hygiene-gate`), then the provider makes the image:
  - **DigitalOcean:** shut the droplet down, take a snapshot, and copy it to each region of `--regions` (default `nyc1,nyc3`). The builder is created in the first region.
  - **GCP:** stop the VM, and make a custom image from its boot disk in the project. A GCP image is global, so it has no region list. The builder is created in the first zone of `--regions`, or in the default zone.
- The builder is destroyed on success and on each failure after it exists, also after Ctrl-C or SIGTERM. Its record and its generated key pair are removed. The build prints what it removed. When the destroy fails, the build prints the command that removes the builder.
- The build writes a record to `images.json` in the spekk config directory: name, provider, provider image ID, regions, project (GCP), created date (RFC 3339), spekk version, OS version (`PRETTY_NAME` of the builder), and the SHA-256 of the setup script (empty with no script). When the provider made an image but a later step failed (for example a region copy), the record is still written, with the regions that have the image, so `image delete` can remove it.

## Verification

- A test with a fake provider runs a build that succeeds: the image record has each field, and the builder is destroyed and its record removed.
- The hygiene test in `image-hygiene-gate` covers the failure path.
- A walkthrough on a local fake cloud (a fake `gcloud` that runs Docker containers) runs `image build` from start to finish.
