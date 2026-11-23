---
slug: artifact-upload-assertion-says-prints-app-link
type: outdated_specs
severity: medium
status: resolved
created: 2026-11-23T00:00:00Z
announced: 2026-11-23T20:01:44Z
affected:
  - specs/agent-file-delivery/assertions/artifact-upload-command.md
  - specs/agent-file-delivery/assertions/artifact-direct-link.md
  - specs/agent-file-delivery/assertions/text-artifact-command.md
  - cmd/spekk/artifact.go
---

# Artifact upload assertion still says the command prints the app link

## Issue Description

The base assertion `artifact-upload-command` (status `done`) says that `spekk artifact upload` "prints the app link" and that "Kind defaults from a known file extension." Later assertions in the same spec changed both behaviors, and the base assertion was not updated. The command now prints the direct `view_url` first and the app page second. It also infers `text` from file content when the extension is unknown. The spec now holds two `done` assertions that disagree about what the command prints.

## Evidence

- `specs/agent-file-delivery/assertions/artifact-upload-command.md`: "It prints the app link. ... Kind defaults from a known file extension."
- `specs/agent-file-delivery/assertions/artifact-direct-link.md` (added in #363): "When the upload answer has `view_url`, `spekk artifact upload` prints it as the first line ... The second line is `App page: <app_url>`". The app link alone is only the older-app fallback.
- `specs/agent-file-delivery/assertions/text-artifact-command.md` (#365): "An unknown extension infers `text` only when its bytes are valid UTF-8 without NUL". This inference is from content, not from the extension.
- `cmd/spekk/artifact.go`: `execArtifact` prints `view` first, then `App page: <app>`. For an unknown extension, its `default:` branch calls `isUTF8File(path)` to infer `text`.

## Impact

A builder or reviewer who reads the base assertion alone gets the old contract: print the app link only. A rework of that assertion could undo the direct-link behavior, and that link is the one agents are told to share. The conflict also makes the spec harder to read: the reader must know which of two `done` assertions takes precedence.

## Recommendation

Update `artifact-upload-command` to describe the current output. It should defer to `artifact-direct-link` for which links are printed and to `text-artifact-command` for kind inference, so that no criterion is stated in two places.