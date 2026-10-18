---
id: image-setup-stdin-closed
parent: sandbox-template-images
created: 2026-10-18T00:00:00Z
priority: 2
status: not_started
depends-on: image-build
branch: feature/sandbox-template-images
---

# A Setup Step That Reads Stdin Does Not Skip the Rest of the Setup

## Success Criteria

- The build sends the setup script in `{ ... } </dev/null`, as `runSecretScript` does. Bash reads the full script first, and each command gets `/dev/null` as stdin.
- A setup that runs `cat >/dev/null` and then a later step runs the later step.

## Verification

- A walkthrough on the fake cloud runs a setup that reads stdin and then touches a file. The image has the file.
