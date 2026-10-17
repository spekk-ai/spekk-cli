---
id: machine-registers-with-one-time-token
parent: sandbox-machine-mode
created: 2026-10-17T00:00:00Z
priority: 1
status: not_started
depends-on: machine-stage-lines
---

# The Box Registers With the Registration Token, and the CLI Makes the Agent Token

The server mints a one-time registration token for the launch. The raw agent token must exist only in the CLI process and on the machine.

**Tests:** internal/sandbox/machine_test.go

## Success Criteria

- At the `registering` stage, the CLI makes an agent token (32 random bytes, base64url, 43 characters) and sends `POST /api/agents/register/` to `spekk_host` with `Authorization: Bearer <registration token>` and the body `{"name": "<name>", "sandbox_id": "spekk-<name>-<8 hex characters>", "token": "<agent token>"}`.
- A `201` with a non-empty `id` registers the agent. The record stores the id. The agent token goes to `/etc/spekk/agent.env` on the machine and nowhere else: not to the record, stdout, or stderr.
- Any other answer is `failed` at `registering`. The message has the status code and the `detail` text of the body, and not the token. No later step runs.
- The register request goes through the one function that sets the `Authorization` header (`authorize`), and does not follow a redirect.
- The register request carries no cloud credential, model credential, or GitHub token, in its URL, its headers, or its body.

## Verification

- The `online` test of `machine-stage-lines` checks the method, path, header, and body of the register request, and that the agent token in the body is the one in the env file that reached the machine.
- One test answers `401` and checks the `failed` line at `registering` and that no equip step ran.
