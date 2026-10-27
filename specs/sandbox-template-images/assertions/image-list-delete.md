---
id: image-list-delete
parent: sandbox-template-images
created: 2026-10-17T00:00:00Z
priority: 2
status: done
depends-on: image-build
branch: feature/sandbox-template-images
---

# `spekk sandbox image list` and `image delete` Keep the Images in Order

**Tests:** internal/sandbox/gcp_provider_test.go (provider delete); the walkthrough covers the commands

## Success Criteria

- `spekk sandbox image list` prints one row for each record in `images.json`: name, provider, provider image ID, regions, created date, age in days, spekk version, and OS version. With no record, it prints `No images.`
- `spekk sandbox image delete <name>` deletes the provider image, then removes the record. When the provider says the image is not there, it prints a warning and removes the record. Any other provider error keeps the record and fails.
- An unknown name fails with an error that names `spekk sandbox image list`.
