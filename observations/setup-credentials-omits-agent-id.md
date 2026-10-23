---
slug: setup-credentials-omits-agent-id
type: outdated_specs
severity: medium
status: open
created: 2026-10-23T18:00:00Z
affected:
  - infrastructure/sandbox/setup-credentials.sh
  - specs/agent-repository-preparation/assertions/every-host-knows-its-agent-id.md
  - specs/agent-repository-preparation/assertions/git-failure-shows-cause.md
  - cmd/sandbox/repository.go
---

# `setup-credentials.sh` never writes `SPEKK_AGENT_ID`, and no spec asks it to

## Issue Description

`every-host-knows-its-agent-id` (#311) fixed repository chat failing with "agent or session ID is missing" by making `create`, `provision`, and `register` write `SPEKK_AGENT_ID` into `/etc/spekk/agent.env`. Its criteria list only those three Go paths. `infrastructure/sandbox/setup-credentials.sh` is a fourth writer of that file. `git-failure-shows-cause` (#312) names it as one, and its header says it injects secrets into "a freshly provisioned sandbox droplet". The script never prompts for or renders `SPEKK_AGENT_ID`. A droplet credentialed from scratch with this script therefore has the defect that #311 fixed for the Go paths, and #312 has just made the same script give the file to `agent`.

When an env file already exists, the line survives a re-run: `load_existing` passes unknown keys through as `EXTRA_LINES`. That only works by accident, because `SPEKK_AGENT_ID` is not in `KNOWN_VARS`. The fresh path has no way to get the ID.

## Evidence

- `infrastructure/sandbox/setup-credentials.sh:81-104`: `render_agent_env` prints `SPEKK_AGENT_TOKEN`, `SPEKK_HOST`, `WORKSPACE`, `SPEKK_AGENT_NAME`, and the mode credential. It prints no `SPEKK_AGENT_ID`.
- `infrastructure/sandbox/setup-credentials.sh:123`: `KNOWN_VARS="GITHUB_TOKEN SPEKK_HOST SPEKK_AGENT_TOKEN WORKSPACE SPEKK_AGENT_NAME"`.
- `infrastructure/sandbox/setup-credentials.sh:229-242`: `collect_credentials` prompts for the agent token but not the agent ID.
- `infrastructure/sandbox/setup-credentials.sh:3`: "Inject secrets into a freshly provisioned sandbox droplet".
- `cmd/sandbox/repository.go:43-44`: `requestRepositoryAccess` fails with "agent or session ID is missing" when `cfg.AgentID` is empty. `cmd/sandbox/client.go:353` takes it only from `SPEKK_AGENT_ID`. The Git credential helper (`cmd/sandbox/git_credential.go:107,161`) also needs it from the file.
- `specs/agent-repository-preparation/assertions/every-host-knows-its-agent-id.md`: the success criteria name `create`, `provision`, and `register` only.
- `specs/agent-repository-preparation/assertions/git-failure-shows-cause.md`: "Each writer of `/etc/spekk/agent.env` in this repository ... `setup-credentials.sh`, the inject script of `create`, `provision`, and machine mode, and the token script of `register`."
- `internal/sandbox/setup_credentials_test.go` has no `SPEKK_AGENT_ID` check.
- A related stale claim from the same change: the comment at `cmd/sandbox/client.go:82-83` still says "A terminal box from spekk sandbox create has no SPEKK_AGENT_ID", but `create` has written it since #311.

## Impact

When an operator follows the script's documented fresh-droplet use, they get a sandbox that connects but fails every repository chat turn. The start warning added in #312 does not catch it, because that warning checks only that the file can be read. An existing file keeps the line only because of the generic pass-through, and nothing tests that a mode switch keeps `SPEKK_AGENT_ID`.

## Recommendation

The remedy commit adds an assertion, `setup-credentials-writes-agent-id` (status `not_started`), under `agent-repository-preparation`. It asks the script to render `SPEKK_AGENT_ID` as a known variable, to take it from the environment or a prompt on a fresh run, to carry it over on a switch, and to have a test for both paths. If a reviewer decides that the script is only for moving a live sandbox, drop the remedy. Instead, change the script header and `every-host-knows-its-agent-id` to say that a fresh droplet gets its ID only from `create`, `provision`, or `register`.
