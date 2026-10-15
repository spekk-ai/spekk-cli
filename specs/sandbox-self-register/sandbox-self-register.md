---
id: sandbox-self-register
created: 2026-10-15T00:00:00Z
priority: 1
---

# A Sandbox Registers Itself, and the CLI Manages Schedules

## Problem

`spekk sandbox create` makes an agent token, writes it to the machine, and prints it. A person must then register the agent on the control host by hand, and must remove it by hand when the sandbox goes away. Scheduled tasks for the agent are also managed only on the control host. A user who has a spekk API token but no admin access cannot finish the life of their own sandbox.

## Solution

The CLI calls the control host API with the user's own API token:

- `create` registers the agent before it provisions the machine, and puts the token that the control host returns on the machine.
- `destroy` revokes the agent before it deletes the machine.
- `register` registers a sandbox that was created without an API token.
- `spekk schedule` lists, creates, updates, deletes, and runs the scheduled tasks of an agent.

## Security rule

The control host never receives a cloud credential or a model credential of the user. The DigitalOcean token, the AWS keys, the Google Cloud credentials, the Claude subscription token, and the GitHub token go from the user's machine to the user's own sandbox, and nowhere else. The control host sees only two credentials: the user's spekk API token, which authenticates each call, and the agent token, which the control host makes.

## Contract with the control host

All calls use `Authorization: Token <SPEKK_API_TOKEN>`. One function sets this header, so a change to the token scheme changes one place.

| Call | Request | Response |
|------|---------|----------|
| Register | `POST /api/agents/` with `name`, `sandbox_id`, and an optional `organization` | `201` with `id` and `auth_token`. The token is shown one time only |
| Revoke | `POST /api/agents/<id>/revoke/` | `200` |
| List schedules | `GET /api/agents/<id>/schedules/?page=<n>` | `200`, paginated: `results` and `next` |
| Create schedule | `POST /api/agents/<id>/schedules/` | `201` with the schedule |
| Update schedule | `PATCH /api/agents/<id>/schedules/<schedule-id>/` | `200` with the schedule |
| Delete schedule | `DELETE /api/agents/<id>/schedules/<schedule-id>/` | `204` |
| Run schedule | `POST /api/agents/<id>/schedules/<schedule-id>/trigger/` | `202` with `message_id`, or `409` with `detail` when the agent is disabled or not connected |

The writable schedule fields are `slug`, `cron_expression`, `timezone`, `task_text`, `custom_context`, `role`, and `enabled`.

`sandbox_id` must be unique on the control host, and a revoked agent keeps its `sandbox_id`. Thus each registration sends a new value: `spekk-<name>-<8 random hex characters>`. A sandbox that is destroyed and then created again with the same name can register again.

## Rejected

- A config file or a keychain for the API token. The environment variable is enough, and a stored token is one more place for a credential to leak.
- A rename or an update of the agent from the CLI. The control host has `PATCH /api/agents/<id>/`, but no step of the sandbox life needs it.

## Assertions

1. [control-host-client](assertions/control-host-client.md) - The API token and the host come from the environment, and no credential reaches a log or stdout
2. [record-stores-agent-id](assertions/record-stores-agent-id.md) - The sandbox record stores the agent id, and no credential
3. [create-registers-agent](assertions/create-registers-agent.md) - `create` registers before it provisions, and revokes on failure
4. [create-without-api-token](assertions/create-without-api-token.md) - `create` without an API token keeps the manual path
5. [register-command](assertions/register-command.md) - `spekk sandbox register <name>` registers a sandbox later
6. [provision-keeps-agent](assertions/provision-keeps-agent.md) - `provision` keeps the agent that the sandbox already has
7. [destroy-revokes-first](assertions/destroy-revokes-first.md) - `destroy` revokes the agent before it touches the machine
8. [schedule-commands](assertions/schedule-commands.md) - `spekk schedule list|create|update|delete|run <agent>`
9. [no-cloud-credential-to-control-host](assertions/no-cloud-credential-to-control-host.md) - No request to the control host carries a cloud or model credential
10. [docs-self-register](assertions/docs-self-register.md) - The CLI reference and the configuration page describe the new behavior
