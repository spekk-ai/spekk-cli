---
icon: lucide/cloud
---

# App-managed sandboxes from a local CLI

Sign in to the app, then open **Settings > API keys > Create key**. Give the key a name and an expiry, and copy its value once. Store it for this CLI with an exact app origin. The command reads the key from standard input and saves it in a user-only file under your local config directory. Do not put the key in a command argument or a repository file.

```bash
read -rs SPEKK_PERSONAL_KEY
printf '%s\n' "$SPEKK_PERSONAL_KEY" | spekk app auth set --host https://app.spekk.ai
unset SPEKK_PERSONAL_KEY
```

The host must use HTTPS. Plain HTTP works only for a loopback development host. The CLI refuses redirects and sends the key only to the exact host. Revoke the key on the API keys page when you no longer need it. If it expires or is revoked, the next API call returns 401; create a replacement key and run `spekk app auth set` again. The CLI stores the key in `$XDG_CONFIG_HOME/spekk/app-auth.json`, or the operating system's default config directory, with mode 0600.

These commands use the app API. `spekk sandbox` manages cloud machines directly and can need provider credentials. `spekk conversation open` runs inside a sandbox session. Neither is part of this app-managed flow.

## Launch, message, and clean up

```bash
spekk app sandbox options
spekk app sandbox repositories --organization ORG_ID --search my-repo
spekk app sandbox launch --organization ORG_ID --audience team --size SIZE --region REGION --kind one-off --repository REPO_ID
spekk app sandbox wait --launch LAUNCH_ID --timeout 10m
spekk app sandbox send --launch LAUNCH_ID --text 'Run the tests'
spekk app sandbox result --agent AGENT_ID --message MESSAGE_ID --timeout 10m
spekk app sandbox send --agent AGENT_ID --text 'Explain the failure' --session SESSION_ID
spekk app sandbox destroy --launch LAUNCH_ID
spekk app sandbox status --launch LAUNCH_ID
```

Use `--audience personal` for a personal sandbox. Its owner must sign in to Claude Code on the remote machine before tasks can run. A team sandbox uses the organization's model credential and needs an organization admin to launch it. The server decides current organization membership, agent grants, repository push access, cloud funding, size and organization limits, expiry, and launch flags. An app admin role does not grant GitHub push access. A repository-free launch needs an explicit organization and works only when the app enables its repository-free worker path. The CLI never receives the model, cloud, GitHub App, SSH, or coordinator credential.

An organization member with an agent write grant can use `spekk app sandbox agents` to find an app-managed agent and send with `--agent`. The app checks the grant again for each send. A member cannot use a grant to change another organization's sandbox. The app launch visibility rule controls cancel and destroy.

`send` returns the launch, agent, request, and message IDs promptly. `result` reads the saved reply and Claude session ID. Give that session ID to a later `send` only for the same agent and a conversation the app permits you to continue. Text works without file attachments or result artifacts. Those features need their separate app contracts and deployment settings. Use `--json` on each command for one JSON object with stable ID, state, reply, and error fields.

## Recover after a stop or timeout

The CLI saves a launch or send request ID before network I/O. `spekk app sandbox operations` lists these local records after a restart. If a launch response is lost, use `spekk app sandbox status --request-id REQUEST_ID`. If a send response or stream is lost, use `spekk app sandbox result --agent AGENT_ID --request-id REQUEST_ID`. A task continues after the local command stops. Reading saved status does not send another task or start another sandbox.

If the lookup shows a saved request whose dispatch did not complete, you can run the same launch or send command with `--request-id REQUEST_ID` and the same fields. The CLI checks its local operation record and the app checks the request body. If the lookup cannot find the request, the outcome is uncertain; choose whether to make a manual retry with that same ID. A changed body gets 409.

Cancel and destroy have no request-ID lookup. Use the launch ID to read `status` after an uncertain response. The CLI does not repeat a cleanup POST automatically. Use `cancel` only before the launch is online and `destroy` only when it is online. A local process exit does not cancel a task or a billable machine; choose cleanup explicitly.

Cleanup is complete only when the launch stage is `failed` or `destroyed` and `finished` is set. A failed launch with `finished` null can still cost money. An online launch can have `finished` set and still cost money. The CLI warns until cleanup is complete. The app worker and orphan sweep own automatic cleanup after failure. Monitor a stalled launch and ask an organization admin to resolve it.

The first integration uses REST and the app's saved status route. An optional MCP adapter can call the same client for options, repositories, launch and status, send and result, and cleanup. It needs no provider or policy layer and no long MCP wait. The current text workflow does not need WebSocket or an MCP server.

## Fresh home terminal example

This transcript uses a loopback API fixture with fake organization, repository, agent, and reply data. The key is entered without terminal echo. The fixture does not launch a cloud machine or run a model.

```text
$ env -i HOME=/tmp/spekk-fresh-home-lc2-final PATH=/usr/bin:/bin /tmp/spekk-local-cli app auth set --host http://127.0.0.1:8765
App host and personal key saved in a private file.
$ env -i HOME=/tmp/spekk-fresh-home-lc2-final PATH=/usr/bin:/bin /tmp/spekk-local-cli app sandbox options --json
{"organizations":[{"id":"org-1","is_admin":true,"name":"Example"}],"regions":["east"],"repository_free_enabled":false,"sizes":[{"slug":"small"}]}
$ env -i HOME=/tmp/spekk-fresh-home-lc2-final PATH=/usr/bin:/bin /tmp/spekk-local-cli app sandbox launch --organization org-1 --audience team --size small --region east --kind one-off --repository repo-1 --json
{"agent_id":"agent-1","error":"","expires_at":"2026-10-25T04:00:00Z","finished":"2026-10-23T04:00:00Z","launch_id":"launch-1","request_id":"70fd774bd8f51bb7784ee7147ab47f69","stage":"online","state":"online","warning":"Cleanup is incomplete. A machine can still cost money; monitor the launch and ask an organization admin if cleanup stalls."}
$ env -i HOME=/tmp/spekk-fresh-home-lc2-final PATH=/usr/bin:/bin /tmp/spekk-local-cli app sandbox send --launch launch-1 --text 'Run tests' --json
{"agent_id":"agent-1","agent_session_id":null,"error":null,"launch_id":"launch-1","message_id":"message-1","reply":null,"request_id":"ea1b655392f5992993277405a9fe878b","session_id":null,"state":"pending"}
$ env -i HOME=/tmp/spekk-fresh-home-lc2-final PATH=/usr/bin:/bin /tmp/spekk-local-cli app sandbox result --agent agent-1 --message message-1 --json
{"agent_id":"agent-1","agent_session_id":"agent-session-1","error":"","launch_id":"launch-1","message_id":"message-1","reply":"Tests passed.","request_id":"ea1b655392f5992993277405a9fe878b","session_id":"claude-session-1","state":"completed"}
$ env -i HOME=/tmp/spekk-fresh-home-lc2-final PATH=/usr/bin:/bin /tmp/spekk-local-cli app sandbox destroy --launch launch-1 --json
{"agent_id":"agent-1","error":"","expires_at":"2026-10-25T04:00:00Z","finished":"2026-10-23T04:30:00Z","launch_id":"launch-1","request_id":"70fd774bd8f51bb7784ee7147ab47f69","stage":"destroyed","state":"destroyed"}
```
