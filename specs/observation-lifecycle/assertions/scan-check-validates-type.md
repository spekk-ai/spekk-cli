---
id: scan-check-validates-type
parent: observation-lifecycle
created: 2026-09-09T12:00:00Z
priority: 1
status: done
---

# Scan Check Rejects Unsupported Observation Types

## Description

`spekk observer scan-check` accepts the same observation types as the file parser. A successful check cannot approve a type that the parser rejects. This fixes the type-validation defect in issue #203.

## Success Criteria

- `code_spec_misalignment` and `outdated_specs` pass type validation and continue to suppression and duplicate checks.
- An unsupported `--type` exits with a nonzero status before the command reads suppression files or observation branches.
- The error identifies `--type`, its rejected value, and the accepted values. The command writes no successful JSON result.
- The command and the file parser use one type-validation rule.

## Verification

`cmd/spekk/observer_test.go` checks rejection before Git reads and accepted types through the scan-check command. `internal/observation/observation_test.go` checks file-parser validation.
