---
id: cli-v1-51-1-release-notes
parent: release-notes-v1-51-1
created: 2026-11-20T00:00:00Z
priority: 2
status: done
branch: docs/v1.51.1
---

# Operators Can Read the 1.51.1 Change

The release notes state which artifact link the CLI prints first and what an older app returns.

## Success Criteria

- The release note, release notes index, and site navigation include 1.51.1.
- The note states that `spekk artifact upload` prints `view_url` on the artifact host as the first plain URL and `App page: <app_url>` as the second line when both links are present.
- The note states that an app without `view_url` causes the CLI to print `app_url` alone, with no label.
- The note states that the direct link needs spekk-app #718, deployed 2026-11-20, and that the protocol stays `1.6`, with major `1`.
- Current documentation does not describe the 1.51.0 output as the 1.51.1 output.

## Verification

- Compare each claim to source, validate the specs, and build the documentation site.
