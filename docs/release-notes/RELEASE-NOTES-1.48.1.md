# Spekk CLI 1.48.1 - No GitHub Token in ~/.git-credentials

This patch release is a security fix. Sandbox credential setup no longer writes the GitHub token to `/home/agent/.git-credentials`. Git gets the token from a credential helper. There are no new commands or flags, and the sandbox protocol does not change.

## Git gets the token from the GitHub CLI

Two paths set up the Git credentials of the `agent` user on a sandbox, and both now do the same steps:

- the git step of `spekk sandbox create` and `spekk sandbox provision`, which the CLI shows as "Configuring git credentials..."
- `infrastructure/sandbox/setup-credentials.sh`, which you run on the machine

Each path signs `agent` in to the GitHub CLI with `gh auth login --with-token`, and checks the login with `gh auth status`. Then it removes each global `credential.helper` entry and each global `credential.https://github.com.helper` entry of `agent`, and sets the helper for `https://github.com` to `gh auth git-credential`. When Git needs a credential for GitHub, the helper reads the GitHub CLI login. The GitHub CLI keeps the token in its login, as before. Git no longer keeps a second copy in a Git credential file. Before this release, each path set `credential.helper store` and wrote the token to `~/.git-credentials`.

## Setup removes the old token file

After it sets the helper, each path asks Git for a GitHub credential as `agent`. When Git returns a password, the path removes `/home/agent/.git-credentials`. Thus a sandbox that an earlier version set up loses the old file when you run credential setup again. The path removes the file only after the lookup succeeds, so a failed setup keeps the file.

## Setup stops when the lookup fails

When the helper lookup returns no password, the path prints `spekk: GitHub credential helper did not return a password` and exits with a failure. Steps that ran before it stay done. For example, the env file `/etc/spekk/agent.env` is already written.

- `create` and `provision` stop at the step "configuring git credentials" and do not deploy the agent.
- `setup-credentials.sh` exits with status 1.

The script also stops now when the GitHub CLI login fails. Before this release, the script ignored a failed `gh auth login`. Now a failed login stops the script, and a failed `gh auth status` prints `ERROR: the agent user is not logged in to GitHub` and exits with status 1. The CLI path already stopped when `gh auth status` failed.

Machine mode (`spekk sandbox create --machine`) does not use these paths, and its credential helper does not change.

## What you must do

An existing sandbox keeps its `~/.git-credentials` file until credential setup runs again. You do not change a configuration value. Do one of these steps for each sandbox:

- Update spekk to 1.48.1 and run `spekk sandbox provision <name> --force`. This runs the credential setup again and deploys the 1.48.1 agent.
- Copy the 1.48.1 `infrastructure/sandbox/setup-credentials.sh` to the machine and run it again. See [Updating sandbox variables](../configuration.md#updating-sandbox-variables).
- Replace the sandbox with a new one that spekk 1.48.1 creates.

## Compatibility

There is no protocol change. The protocol version stays `1.6`, and the protocol major stays `1`. An agent at 1.48.1 and a control host at any `1.x` version connect as before.
