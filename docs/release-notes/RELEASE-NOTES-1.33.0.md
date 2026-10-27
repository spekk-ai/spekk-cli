# Spekk CLI 1.33.0 - A Sandbox Registers Itself, and Sees Its Background Tasks End

Before this release, you registered each sandbox agent on the control host by hand, and `destroy` left that agent live. Scheduled tasks had no command in the CLI. On a sandbox from cloud-init or the AWS template, Claude Code came from npm at version 2.1.197, which stops a background task a few seconds after the turn gives its result. The agent then never saw how that task ended.

This release registers the agent through the control host API with your own API token, revokes it on `destroy`, and adds `spekk schedule`. Every sandbox now runs Claude Code 2.1.292 or newer. The `result` frame tells the control host if a turn ended its work or only yielded. `spekk observer --mcp-config` now gives its MCP servers to Claude Code.

## A sandbox registers itself

Set `SPEKK_API_TOKEN` to your spekk API token from the control host. Then:

- `spekk sandbox create` registers the agent on the control host before it asks the provider for a machine, so a refused registration costs nothing. It writes the agent token that the control host returns to `/etc/spekk/agent.env` on the machine, and stores only the agent id in the local record. When a later step fails, or when you press Ctrl-C, `create` revokes the agent again. On success, it prints the agent id, not the token.
- `spekk sandbox provision` keeps the agent of a registered sandbox: it writes the same token back, so the agent keeps its schedules and its history. A sandbox with no agent id gets a new agent token, which `provision` registers when `SPEKK_API_TOKEN` is set.
- New `spekk sandbox register <name>` registers a new agent for a recorded sandbox, writes its token to the machine, and restarts the agent. If a step fails, it puts the old token back. Use it for a sandbox that you created without `SPEKK_API_TOKEN`. A new agent starts with no schedules.
- New `--organization <id>` on `create`, `provision`, and `register`. You need it only when you belong to more than one organization. Without it, the control host refuses the registration, and the error tells you to give the flag.

Without `SPEKK_API_TOKEN`, `create` operates as before: it makes the token, prints it, and you register it by hand.

The control host receives only two credentials: your spekk API token and the agent token that it makes itself. Your cloud, model, and GitHub credentials go from your machine to your sandbox over SSH, and nowhere else. The CLI sends the `Authorization` header only to `SPEKK_HOST`, over `https`. Plain `http` is accepted only for `localhost` and a loopback address.

## `destroy` revokes the agent first

When the record has an agent id, `spekk sandbox destroy` revokes the agent on the control host before it touches the machine. This needs `SPEKK_API_TOKEN` and `SPEKK_HOST`. Without them, `destroy` stops before the confirmation prompt. When the revoke fails, `destroy` stops, and the machine and the record stay.

`--force` only skips the prompt. It does not skip the revoke. New `--ignore-revoke-failure` continues after a failed revoke or a missing `SPEKK_API_TOKEN`, and prints a warning with the agent id. Then revoke that agent on the control host yourself.

A sandbox that has no agent id in its record, for example every sandbox created before this release, is destroyed as before.

## `spekk schedule`

New `spekk schedule list|create|update|delete|run` manages the scheduled tasks of a sandbox agent on the control host:

```bash
spekk schedule create my-sandbox --slug nightly-scan --cron_expression "0 9 * * *" --task_text "Scan the specs for drift"
spekk schedule run my-sandbox nightly-scan
```

The agent is a sandbox name with an agent id in its record, or an agent id. A schedule is its id or its slug. Every subcommand needs `SPEKK_API_TOKEN` and `SPEKK_HOST`. The [CLI reference](../cli-reference.md#spekk-schedule) lists the fields.

## Claude Code 2.1.292 or newer on every sandbox

The cloud-init template and the AWS template now install the native Claude Code from the `stable` release as the `agent` user, as `scripts/prepare-machine.sh` already did. The installer also keeps the self-updater on `stable`. The minimum version is 2.1.292. An older version stops a background task a few seconds after the `result` and exits, so the agent never sees how the task ended. From 2.1.292, Claude Code keeps the `claude -p` process alive while a background task runs, and reports again when the task ends.

- `spekk sandbox deploy` reads the version of Claude Code that the service runs. When it is older than the minimum, or cannot be read, the deploy installs the `stable` release and reads the version again. When the version is still older, the deploy fails and does not restart the agent.
- Before each turn, the agent reads the version again. On a version older than the minimum, it does not start the turn. It sends an `error` frame that names the version, the minimum, and the remedy: run `spekk sandbox deploy`.
- The checks fail open. When the agent or the deploy cannot read a version, it gives a warning and continues. Only a version that was read and is older than the minimum stops a turn or a deploy.

## Protocol 1.1: the result frame says if the turn is terminal

The agent now declares protocol `1.1`. The only change is a new `terminal` field, a JSON boolean, on each `result` frame. It is `false` when a background task that the turn started was still live at the last `result` event of Claude, so the turn only yielded. It is `true` in all other cases. The agent reads this from structured fields of the Claude stream only, never from the words of the agent.

The change is additive. A control host that speaks `1.0` has the same major version, so it accepts a `1.1` agent, and the agent shows no warning. A control host reads an absent `terminal` field as `true`, so an old agent and a new control host also work together. Either side can update first. The `turn_keepalive` frame type is reserved for protocol `1.2`, and this agent does not send it. See [Sandbox Agent Architecture](../advanced/sandbox-architecture.md#turn-progress).

## `spekk observer --mcp-config`

Before this release, `spekk observer` accepted `--mcp-config <file>` and did not give it to Claude Code. The run started without those MCP servers, and showed no error. Now the observer gives the file to Claude Code for that run only. This applies to a headless run and to an interactive run, and to the default scan and to a skill:

```bash
spekk observer my-skill --headless --mcp-config browser-mcp.json
```

Use it to give one scheduled skill a tool that no other session loads, for example a browser. A missing value, or a path that is not a file, stops the run before Claude Code starts. See the [CLI reference](../cli-reference.md#spekk-observer).

## Upgrading

1. Run `spekk update` to install the 1.33.0 CLI.
2. Run `spekk sandbox deploy <name>` for each sandbox. The deploy installs the new agent, and it upgrades Claude Code when the version is older than 2.1.292. Then it restarts the agent service, and this stops a turn that is running. Deploy when no turn runs. A sandbox that you do not deploy keeps its old agent.
3. To manage an existing sandbox through the API, set `SPEKK_API_TOKEN` and run `spekk sandbox register <name>`. The sandbox gets a new agent with no schedules. The agent that you registered by hand before stays live, so revoke it on the control host. `spekk sandbox provision` with `SPEKK_API_TOKEN` set also registers a new agent for a sandbox that has no agent id.

Your sandbox records need no change. A record with no agent id operates as before, and `spekk schedule` accepts an agent id for it.
