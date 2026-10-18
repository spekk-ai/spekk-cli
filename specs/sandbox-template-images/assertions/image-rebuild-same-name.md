---
id: image-rebuild-same-name
parent: sandbox-template-images
created: 2026-10-18T00:00:00Z
priority: 1
status: done
depends-on: image-build
branch: feature/sandbox-template-images
---

# A Rebuild Under the Same Name Does Not Use the Old Host Key

**Tests:** internal/sandbox/image_test.go

## Success Criteria

- The build removes the known_hosts file of the builder (`KnownHostsFile("image-<name>")`) before it creates the builder, and `destroyBuilder` removes it with the builder record and key pair.
- After `image delete X`, `image build --name X` connects to the new builder without a host key error, and does not wait for the provisioning timeout.

## Verification

- A test plants an old known_hosts file for the builder. The create sees no file, and after the build there is no file. The test fails on the code before this assertion.
- A walkthrough on the fake cloud builds an image, deletes it, and builds it again under the same name.
