---
id: auth-flag-selects-mode
parent: sandbox-auth-mode
created: 2026-08-27T00:00:00Z
priority: 1
status: done
branch: feat/sandbox-subscription-auth
---

# `--auth` Selects the Credential a Sandbox Is Built With

`spekk sandbox create` takes an `--auth <mode>` flag naming which credential the new sandbox authenticates Claude with. An unknown mode is refused before the command creates anything billable.

**Tests:** internal/sandbox/auth_test.go

## Success Criteria

- `internal/sandbox` defines the mode as a named string type with exactly two values, `bedrock` and `subscription`, and a `ParseAuthMode(string) (AuthMode, error)` that maps the flag's text to one of them.
- `ParseAuthMode("")` returns `bedrock`. An operator who passes no `--auth` flag gets today's behavior.
- `ParseAuthMode` on any other value returns an error naming the value it got and listing both accepted modes, so the message tells the operator what to type next.
- `CreateOptions` (`internal/sandbox/commands.go`) carries the mode in an `Auth` field.
- `createSandbox` in `cmd/spekk/main.go` registers `--auth` in its `cli.FlagSet` as a `cli.StringFlag`, parses it through `ParseAuthMode`, and on error prints `Error: <message>` to stderr and exits 1.
- The `--auth` line appears in the `spekk sandbox create --help` text with its two values and its default.

**Note:** the rejection must happen in `createSandbox`, before `sandbox.Create` runs. `Create` uploads an SSH key and creates a droplet; a mode typo caught after that point costs the operator a half-built sandbox to clean up by hand.

## Verification

- `go test ./internal/sandbox` covers `ParseAuthMode` for: empty input, both valid modes, and one unknown value whose error text names the value and both modes.
- `go build ./cmd/spekk && ./spekk sandbox create --help` shows the `--auth` line.
- `./spekk sandbox create --name probe --auth nonsense` exits 1 with the mode error and creates nothing. Run it with no cloud credentials set, so a regression that reaches `Create` fails on a different message and is visible as such.
