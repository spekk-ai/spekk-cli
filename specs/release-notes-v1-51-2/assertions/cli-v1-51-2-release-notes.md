---
id: cli-v1-51-2-release-notes
parent: release-notes-v1-51-2
created: 2026-11-20T00:00:00Z
priority: 2
status: done
branch: docs/v1.51.2
---

# Operators Can Read the 1.51.2 Changes

The release notes state what `spekk artifact upload` accepts and refuses at tag v1.51.2.

## Success Criteria

- The release note, release notes index, and site navigation include 1.51.2. The note attributes no change after tag v1.51.2 to this release.
- The note states that known text and code extensions infer `text`, that an unknown extension infers `text` only for a file of at most 1 MiB with valid UTF-8 and no NUL bytes, and that `--kind text` selects the kind explicitly.
- The note states that HTML, HTM, SVG, and XHTML files fail inference with `cannot infer the artifact kind; active content is not a text artifact`, while unknown binary files and unknown files over 1 MiB fail with `cannot infer the artifact kind; use --kind`.
- The note states that an app response of HTTP 400 with `Choose the kind` or `Unknown kind` for kind `text` causes `this app does not support text artifacts yet: <app response>`; HTTP 404 reports that artifacts are off.
- The note states that a successful upload prints the direct content link first and `App page: <app_url>` second when both URLs exist, and that kind `text` needs the Spekk app release with spekk-app #720, deployed 2026-11-20.
- The note states that the WebSocket protocol stays `1.6` with major `1`. Current CLI reference text describes the inference limits and refusal paths accurately.

## Verification

- Compare each claim to the code at tag v1.51.2, validate the specs, and build the documentation site.
