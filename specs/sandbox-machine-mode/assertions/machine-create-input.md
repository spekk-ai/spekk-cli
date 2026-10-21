---
id: machine-create-input
parent: sandbox-machine-mode
created: 2026-10-17T00:00:00Z
priority: 1
status: not_started
branch: feature/repository-free-machine-launch
depends-on: secrets-on-ssh-stdin
---

# `create --machine` Reads One Strict JSON Object on Stdin

The worker gives each secret of a launch to the CLI on stdin, and on stdin only. A required field that is missing, or an unknown field, is a bug in the caller, and it must stop the launch before anything costs money.

**Tests:** internal/sandbox/machine_test.go

## Success Criteria

- `spekk sandbox create <name> --machine --size <size> --region <region>` takes the name as its first argument (`--name <name>` also works). `--size` and `--region` are required in machine mode.
- Machine mode accepts these flags only: `--size`, `--region`, `--one-off`, `--expires`, `--label`, `--release`, `--provision-timeout`. Any other flag (for example `--provider gcp`, `--ip`, `--auth`, `--organization`) is an error before stdin is read.
- `--label <key>=<value>` can repeat. The key and the value must be lowercase `[a-z0-9_-]`, 1 to 63 characters. A key that spekk sets itself (`spekk-kind`, `spekk-expires`, `spekk-host`, `spekk-sandbox`) is refused. The labels go on the droplet as DigitalOcean tags `<key>:<value>`, with the kind labels.
- The CLI reads stdin to its end, at most 64 KiB, and decodes one JSON object with the fields of the parent spec. A field that is not in the contract, at the top level or in `model_credential`, is an error. The GitHub fields are an optional pair. When both `github_repository` and `github_token` are empty or omitted, the launch has no repository. If either field has a value, both fields must be non-empty strings. Each other required field, and each field of the chosen `model_credential` type, must be a non-empty string. A field of another type (for example `api_key` with `"kind": "bedrock"`) is an error. Data after the object is an error.
- A supplied `github_repository` must be `<owner>/<repository>`, each part `[A-Za-z0-9._-]+`, and neither part `.` or `..`. `spekk_host` follows the rules of `controlHostBase`: `https`, or `http` for a loopback host only.
- `harness` must be `claude-code`. Any other value is an error that names it.
- `model_credential.kind` is `none`, `anthropic_api_key`, or `bedrock`. Any other value is an error that names the three values.
- Each error above writes `{"stage": "failed", "at": "creating", ...}` on stdout, and exits 1. No call to DigitalOcean, the control host, or GitHub happens, and no record or key is written.
- Machine mode reads no secret from its environment: `DO_API_TOKEN`, `DIGITALOCEAN_TOKEN`, `SPEKK_API_TOKEN`, `SPEKK_HOST`, `GITHUB_TOKEN`, `ANTHROPIC_API_KEY`, `CLAUDE_CODE_OAUTH_TOKEN`, and the AWS variables have no effect.
- Without `--release`, machine mode deploys the agent of the release of the CLI itself, so the pinned CLI and its agent match: a CLI with the version `vX.Y.Z` deploys the agent of the tag `vX.Y.Z`. A CLI with any other version (a development build, or an `exp-*` build, whose version has no prefix) needs `--release <tag>`, and without it fails at `creating`. The agent of another release can lack the git credential helper of `machine-git-access`.

## Verification

- One table test feeds stdin cases to the input reader: a valid object of each `model_credential` kind, a missing field, an unknown harness, an empty field, an unknown top-level field, an unknown `model_credential` field, a field of another type, an unknown type, trailing data, a bad repository, an omitted repository and token for each model credential kind, an incomplete GitHub field pair, and an `http` host that is not loopback. Each bad case is an error.
- One test runs `create --machine` with a bad stdin against a fake DigitalOcean server and checks the `failed` line and that the server got no request.
