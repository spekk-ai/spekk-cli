---
slug: machine-contract-omits-provision-timeout
type: outdated_specs
severity: low
status: resolved
created: 2026-11-05T00:00:00Z
affected:
  - specs/sandbox-machine-mode/sandbox-machine-mode.md
  - specs/sandbox-machine-mode/assertions/machine-create-input.md
  - cmd/spekk/sandbox_machine.go
  - docs/cli-reference.md
---

# Machine-Mode Contract Synopsis Omits `--provision-timeout`

## Issue Description
The "Contract with the server" section of `specs/sandbox-machine-mode/sandbox-machine-mode.md` gives the create command line that a server worker runs. It lists `--size`, `--region`, `--one-off`, `--expires`, `--label`, and `--release`, but not `--provision-timeout`. The done assertion `machine-create-input`, the code, and the CLI reference all accept `--provision-timeout` in machine mode. So the parent contract, the part a server author reads, is narrower than the real contract.

## Evidence
- `specs/sandbox-machine-mode/sandbox-machine-mode.md` (Contract with the server): `spekk sandbox create <name> --machine --size <size> --region <region> [--one-off [--expires <when>]] [--label <key>=<value> ...] [--release <tag>]`. It has no `--provision-timeout`.
- `specs/sandbox-machine-mode/assertions/machine-create-input.md`: "Machine mode accepts these flags only: `--size`, `--region`, `--one-off`, `--expires`, `--label`, `--release`, `--provision-timeout`."
- `cmd/spekk/sandbox_machine.go` (`parseMachineCreateArgs`): the flag map has `"--provision-timeout": &timeout`, and the error text lists it among the accepted flags.
- `docs/cli-reference.md` (Machine mode): the synopsis ends with `[--provision-timeout <duration>]`.

## Impact
A server author who works from the parent spec does not learn that the cloud-init wait can be set (default `30m`). A launch on a slow image can then time out with no documented fix. The spec and its own assertion also give two different flag sets.

## Recommendation
Add `[--provision-timeout <duration>]` to the create synopsis in the parent spec, as the CLI reference already shows it. The remedy commit does this.
