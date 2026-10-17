---
id: machine-destroy
parent: sandbox-machine-mode
created: 2026-10-17T00:00:00Z
priority: 1
status: not_started
branch: feature/sandbox-machine-mode
depends-on: machine-create-input
---

# `destroy --machine --force` Reads the DigitalOcean Token on Stdin

**Tests:** internal/sandbox/machine_test.go

## Success Criteria

- `spekk sandbox destroy <name> --machine --force` reads `{"digitalocean_token": "<token>"}` on stdin, with the same strict rules as create: an unknown field, a missing or empty token, or data after the object is an error before any cloud call.
- `--machine` needs `--force`, because stdin carries the token and cannot also answer the prompt. Without `--force`, the command is an error before stdin is read.
- It finds the record under `$HOME`, as a terminal destroy does. A record that is not `digitalocean` is an error.
- It does not revoke the agent: the caller revokes it, and machine mode has no API token. It deletes the droplet and the DigitalOcean SSH key, then the local key pair, the `known_hosts` file, and the record, as a terminal destroy does.
- A droplet that is already gone (`404`) is a warning, and the record is removed, as today.
- It writes nothing to stdout. The exit status is 0 when the record is gone, else 1, with the reason on stderr and no secret.

## Verification

- One test destroys a recorded sandbox against a fake DigitalOcean API with the token on stdin, and checks the two delete requests, their token, that no request went to a control host, and that the record and the key files are gone.
