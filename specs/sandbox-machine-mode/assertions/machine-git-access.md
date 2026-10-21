---
id: machine-git-access
parent: sandbox-machine-mode
created: 2026-10-17T00:00:00Z
priority: 1
status: not_started
branch: feature/repository-free-machine-launch
depends-on: machine-model-credential
---

# The Box Clones Its Repository, and a Git Credential Helper Renews the Token

An installation token lives 1 hour, and a persistent sandbox lives for weeks. The first token clones the repository. After that, git asks the control host for a new token each time it needs one. The spekk-app spec `sandbox-git-access` owns the server endpoint, which does not exist yet.

**Tests:** internal/sandbox/machine_test.go, cmd/sandbox/git_credential_test.go, cmd/sandbox/repository_test.go

## Success Criteria

- When a repository is supplied, at the `registering` stage after the env file is on the machine, the CLI clones `https://github.com/<github_repository>.git` into `/opt/spekk/workspace` as the user `agent`. The script goes on the stdin of `ssh`. The installation token reaches `git` through a pipe and an environment variable of the `git` process, never through an argument, a remote URL, `~/.git-credentials`, or a git config file.
- A clone that fails is `failed` at `registering`.
- When both GitHub fields are empty or omitted, the CLI creates `/opt/spekk/workspace` with owner `agent`, then deploys the agent. It does not clone, create a Git remote, install a Git credential helper, or copy a GitHub credential. The agent can start and run a chat turn in that plain workspace.
- A workspace setup failure is `failed` at `registering`. The launch keeps its droplet and agent record for the caller to destroy and revoke, as the repository launch does. The stage names, one-time registration, model credentials, and expiry rules are the same for both launch modes.
- When a repository is supplied, the CLI sets `credential.https://github.com.helper` to `/opt/spekk/agent-client git-credential` in the global git config of `agent`, and sets `credential.https://github.com.useHttpPath` to `false`. It installs no `store` helper and runs no `gh auth login`.
- `agent-client git-credential get` reads the git credential request on stdin. For `protocol=https` and `host=github.com`, it reads `SPEKK_HOST`, `SPEKK_AGENT_TOKEN`, and `SPEKK_AGENT_ID` from `/etc/spekk/agent.env`, sends `POST /api/agents/<agent id>/git-token/` with `Authorization: Bearer <agent token>`, and prints `username=x-access-token` and `password=<token>` from the `token` field of the answer.
- For any other host or protocol, and for the `store` and `erase` actions, the helper prints nothing and exits 0, so git tries its next helper.
- When the answer is not `200` with a token, the helper prints nothing on stdout, writes one line to stderr with the status code and the `detail` text, and exits 1. Git then shows that line in its error, so a `409` "The GitHub App no longer has access to <repository>" reaches the agent.
- The helper uses `https` for the control host, and `http` for a loopback host only, as the agent client does. It does not follow a redirect.

## Verification

- One test runs the helper against a fake control host: a `200` (it checks the method, the path, the header, and the output), a `409` (it checks the stderr line and the exit status), and a request for another host (it checks that no request went out).
- The `online` test of `machine-stage-lines` checks that the clone script went on stdin and that no `ssh` argument holds the installation token.
