---
id: model-pin-release-notes
parent: release-notes-v1-50
created: 2026-10-28T00:00:00Z
priority: 2
status: done
branch: docs/v1.50.0
---

# Operators Can Read the 1.50.0 Model Pin Change

The release notes describe the model pin that `spekk sandbox create` writes to the sandbox env file.

## Success Criteria

- The 1.50.0 release note, release notes index, and site navigation have an entry for the new version.
- The note states that `--model` sets `ANTHROPIC_MODEL`, with `us.anthropic.claude-opus-5-5` as the Bedrock default and `claude-opus-5-5` as the subscription default.
- The note states that `--model ""` writes no pin and that the caller's `ANTHROPIC_MODEL` does not set the pin.
- The note states that a CLI update or `sandbox deploy` does not change an existing sandbox's env file, while `sandbox provision --force` rewrites it with the auth mode's default pin.
- The note gives the operator action and states that there is no protocol change, protocol version stays `1.6`, and protocol major stays `1`.
- No documentation says that the caller's `ANTHROPIC_MODEL` sets the pin.

## Verification

- Compare each claim to the 1.50.0 source and build the documentation site.
