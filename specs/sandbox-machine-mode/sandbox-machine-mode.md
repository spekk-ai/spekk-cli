---
id: sandbox-machine-mode
created: 2026-10-17T00:00:00Z
priority: 1
---

# A Server Launches a Sandbox Through the CLI in Machine Mode

## Problem

A web app will launch sandboxes for its users. Its worker must not keep a second copy of the sandbox code, so it runs the pinned `spekk` binary. The CLI today is for a person at a terminal: it reads each credential from its environment, prints text for a person, and registers with the personal API key of the user (`specs/sandbox-self-register/`). A worker needs a different interface:

- The worker gives the secrets of one launch to one process. An environment variable or a flag of a child process is readable by other processes of the same user on the host, and a flag is readable by each user in `/proc/<pid>/cmdline`.
- The worker must know the stage of the launch while it runs, and the agent id at the end, without a parse of text for a person.
- The box must register with a one-time registration token that the worker mints for the launch, not with a personal key.
- The box must reach one repository with a GitHub App installation token, which lives 1 hour.

The CLI also has a defect in both modes: `injectCredentials`, `configureGitCredentials`, and `buildTokenScript` put the base64 secret in the `ssh` argument list. Each process on the host can read it in `/proc/<pid>/cmdline` while `ssh` runs.

## Solution

`--machine` on `spekk sandbox create` and `spekk sandbox destroy`.

- **Input.** One JSON object on stdin. Flags carry no secret. Machine mode reads no secret from its environment.
- **Output.** One JSON line on stdout for each stage. Human logs go to stderr. No output holds a secret.
- **Cloud.** DigitalOcean only, with the token from stdin: the token of the DigitalOcean connection of an organization, or of spekk's own account. The `Provider` interface does not change, so another cloud fits later.
- **Harness and model credential.** `harness` is `claude-code`, the only harness now. The model credential is `none`, `anthropic_api_key`, or `bedrock`. Never a Claude subscription token: that token is for one person. With `none`, the person signs in to Claude Code on the box later, and the heartbeat of the agent says `claude_signed_in: false` until then.
- **Registration.** `POST /api/agents/register/` with `Authorization: Bearer <registration token>`. The CLI makes the agent token and sends it in the body.
- **Git.** The CLI clones the repository into the workspace, and installs a git credential helper that asks the control host for a new installation token when git needs one.
- **Secrets on SSH stdin.** Each script that holds a secret goes to the machine on the stdin of `ssh`, in both modes.
- **State under `$HOME`.** The record and the SSH key stay under `$HOME/.config/spekk/`, as today. The worker gives each launch its own `HOME`, so a later destroy finds them.

### Contract with the server

This is the CLI side of the "Contract with spekk-cli" section of the spekk-app spec `sandbox-launch`.

```
spekk sandbox create <name> --machine --size <size> --region <region> [--one-off [--expires <when>]] [--label <key>=<value> ...] [--release <tag>]
spekk sandbox destroy <name> --machine --force
```

Create stdin:

```json
{
  "digitalocean_token": "<token>",
  "registration_token": "<token>",
  "spekk_host": "<host>",
  "github_token": "<installation token>",
  "github_repository": "<owner>/<repository>",
  "harness": "claude-code",
  "model_credential": {"kind": "none"}
}
```

`model_credential` is one of:

```json
{"kind": "none"}
{"kind": "anthropic_api_key", "api_key": "<key>"}
{"kind": "bedrock", "aws_access_key_id": "<id>", "aws_secret_access_key": "<secret>", "aws_region": "<region>"}
```

An optional `"image": "<name>"` names a template image (`specs/sandbox-template-images/`). The CLI finds it by name in the account's private images. Without it, or when the region has no copy, the droplet gets the full cloud-init, and the stage lines are the same.

Destroy stdin: `{"digitalocean_token": "<token>"}`.

Stage lines on stdout, in this order:

```json
{"stage": "creating"}
{"stage": "provisioning", "droplet_id": 123, "ip": "203.0.113.10"}
{"stage": "registering"}
{"stage": "online", "agent_id": "<id>"}
```

or, at any point, `{"stage": "failed", "at": "<stage>", "message": "<text with no secret>"}`.

Calls to the control host:

| Call | Request | Response |
|------|---------|----------|
| Register | `POST /api/agents/register/`, `Authorization: Bearer <registration token>`, body `name`, `sandbox_id`, `token` | `201 {"id": "<agent id>"}` |
| Git token (from the box) | `POST /api/agents/<agent id>/git-token/`, `Authorization: Bearer <agent token>` | `200 {"token", "expires_at"}` |

The server side of both calls does not exist yet. Go tests use a fake server.

## Security rule

There are two paths, and the rule of `specs/sandbox-self-register/` names both:

- **From a terminal.** The control host never receives a cloud credential or a model credential of the user. They go from the user's machine to the user's sandbox only.
- **Machine mode.** The control host keeps the DigitalOcean token and the model key of an organization (sealed), and its worker gives them to the CLI on stdin. The CLI sends them to the machine only, on the stdin of `ssh`, and writes the DigitalOcean token to no file. No request from the CLI to the control host API carries a cloud credential or a model credential: the registration request carries the registration token and the new agent token only.

## Rejected

- **Secrets in environment variables of the CLI.** The environment of a process is readable by each process of the same user, and a child inherits it. Stdin is read one time and is gone.
- **A Claude subscription token in machine mode.** It is the credential of one person, and the server launches for an organization.
- **The CLI revokes the agent when a machine-mode create fails.** The CLI has no API token in machine mode. The server revokes the agent of a failed launch, because it already owns the cleanup of `launch-failure-and-retry`.
- **The CLI destroys the droplet when a machine-mode create fails.** The server runs `destroy --machine` with the same `HOME`, as for a terminal create that fails. One cleanup path is enough.
- **A token cache on the box for the git helper.** One HTTP call for each git operation is cheap. A cache is one more file with a token in it.
- **`--machine` for GCP, AWS, or a machine you already have.** No caller needs them now.
- **New `--auth` values for the terminal path.** `none` and `anthropic_api_key` are for machine mode only. The terminal path keeps `bedrock` and `subscription`.

## Assertions

1. [secrets-on-ssh-stdin](assertions/secrets-on-ssh-stdin.md) - No secret is in the argument list of `ssh`, in either mode
2. [machine-create-input](assertions/machine-create-input.md) - `create --machine` reads one strict JSON object on stdin, and refuses a bad one before any cloud call
3. [machine-stage-lines](assertions/machine-stage-lines.md) - `create --machine` writes one JSON line for each stage, and nothing else on stdout
4. [machine-registers-with-one-time-token](assertions/machine-registers-with-one-time-token.md) - The box registers with the registration token, and the CLI makes the agent token
5. [machine-model-credential](assertions/machine-model-credential.md) - The env file carries the model credential of the chosen kind, and nothing of the other kinds
6. [agent-reports-sign-in-required](assertions/agent-reports-sign-in-required.md) - An agent with no Claude credential reports `claude_signed_in: false`, and refuses turns with a clear message
7. [machine-git-access](assertions/machine-git-access.md) - The box clones its repository, and a git credential helper renews the token through the control host
8. [machine-destroy](assertions/machine-destroy.md) - `destroy --machine --force` reads the DigitalOcean token on stdin
9. [machine-mode-docs](assertions/machine-mode-docs.md) - The CLI reference describes machine mode
