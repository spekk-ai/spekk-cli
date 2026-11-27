---
id: sandbox-remote-serve
created: 2026-11-08T00:00:00Z
priority: 2
---

# Remote coach session on a sandbox

A developer can run `spekk serve` on a sandbox and talk to the coach from a local terminal, without opening a firewall port or installing a WebSocket client. When the sandbox is registered with the control host, the developer can hand the session to the app and continue it there.

Today this takes a manual `spekk sandbox ssh … -L`, a hand-run `spekk serve`, a copied nonce, and `websocat`. The serve process runs as the SSH login user, which cannot read `/etc/spekk/agent.env` (mode 600, owned by `agent`), so it does not use the sandbox's model credential.

## Commands

- `spekk sandbox serve start <name>` — runs serve on the sandbox.
- `spekk sandbox serve connect <name>` — opens an interactive session from the local terminal.
- `spekk sandbox serve status <name>` / `spekk sandbox serve stop <name>`.

The names follow the existing `spekk sandbox <verb> <name>` form.

## Design decisions

- Serve runs as the `agent` user with `/etc/spekk/agent.env`, the same user and environment as the worker, as a supervised service that survives the SSH session. It gets the same credentials as the worker; no narrower set.
- Serve binds to localhost only. `connect` reaches it through an SSH tunnel with the sandbox's key. No port opens to the network.
- The nonce and port live in a file on the sandbox that only `agent` and root can read. `connect` reads it over SSH, so the nonce never needs to be copied or printed.
- Serve works in its own directory, never in a checkout the worker prepares for a turn, so the coach and an app turn never write to the same repository.
- The handoff reuses the existing `conversation_open` frame. The control host already binds the conversation to a Claude session and routes replies back as a resume.

## Known limits (documented, not fixed here)

- Serve and the worker share the `agent` user's Claude home (`~/.claude`): settings, hooks, and MCP config changed in one affect the other.
- In subscription mode, coach sessions draw on the same rate limit as the worker's turns.
- A dropped connection ends that coach session unless it was handed to the app; there is no local resume.
- The coach runs with `--dangerously-skip-permissions` and the worker's tokens (`GITHUB_TOKEN`, `SPEKK_AGENT_TOKEN`), as the worker's Claude does.
- A handoff request comes from a process running as `agent`, so the worker cannot prove which local process wrote it. It only accepts session IDs that serve started.

## Assertions

1. [serve-start](assertions/serve-start.md) — serve runs on the sandbox as `agent` with the agent environment.
2. [serve-connect](assertions/serve-connect.md) — an interactive local session through a tunnel.
3. [serve-stop-status](assertions/serve-stop-status.md) — the service can be inspected and stopped.
4. [serve-app-handoff](assertions/serve-app-handoff.md) — a coach session continues in the app.
5. [serve-remote-docs](assertions/serve-remote-docs.md) — the CLI reference covers the commands and limits.
