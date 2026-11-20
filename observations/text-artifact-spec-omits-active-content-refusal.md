---
slug: text-artifact-spec-omits-active-content-refusal
type: outdated_specs
severity: medium
status: resolved
created: 2026-11-20T12:00:00Z
announced: 2026-11-20T20:01:40Z
affected:
  - specs/agent-file-delivery/assertions/text-artifact-command.md
  - cmd/spekk/artifact.go
  - docs/cli-reference.md
---

# Text artifact spec omits the active-content refusal for HTML and SVG

## Issue Description

The `text-artifact-command` assertion says an unknown extension infers `text` whenever its bytes are valid UTF-8 without NUL, and that every other unknown file keeps the current inference error. The CLI does something the assertion does not state: it refuses to infer any kind for `.html`, `.htm`, `.svg`, and `.xhtml`, even when the file is valid UTF-8, and it gives a different error for them. The CLI reference documents that refusal. The assertion, which is the spec, does not.

## Evidence

- `specs/agent-file-delivery/assertions/text-artifact-command.md`, success criterion 3: "An unknown extension infers `text` only when its bytes are valid UTF-8 without NUL; other unknown files keep the current inference error." Read as written, a UTF-8 `.html` file infers `text`.
- `cmd/spekk/artifact.go`, in the `default:` branch of the extension switch in `execArtifact`: `else if ext == ".html" || ext == ".htm" || ext == ".svg" || ext == ".xhtml" { return fmt.Errorf("cannot infer the artifact kind; active content is not a text artifact") }`. This runs before the UTF-8 check. Commit a5a94a9 (#365) added it, in the same change that added the assertion.
- `docs/cli-reference.md` (artifact upload section, about line 1197): "The CLI refuses to infer a kind for `.html`, `.htm`, `.svg`, or `.xhtml` with `cannot infer the artifact kind; active content is not a text artifact`."

## Impact

The refusal looks like a deliberate safety rule: it stops markup that a browser can run from being published by inference. Because only the docs and the code hold it, a builder that reworks inference to match the assertion could remove it and still pass every stated criterion. A reviewer who reads the spec also cannot tell whether the refusal is intended or a bug.

## Recommendation

Add the refusal to the assertion as a success criterion that names the four extensions and the exact error, so the spec, the docs, and the code agree. The remedy commit on this branch makes that edit. If the refusal was not intended, delete this branch and file the code side instead.