---
id: root-does-not-trust-agent-paths
parent: sandbox-spekk-binaries
created: 2026-10-18T06:20:00Z
priority: 1
status: done
branch: feat/sandbox-spekk-binaries
depends-on: deploy-installs-matching-cli
---

# Root Does Not Follow or Run What agent Can Change

## Success Criteria

- `agent` owns `/opt/spekk`, so it can replace `/opt/spekk/cli` with a link. When `/opt/spekk/cli` is a link, the install script removes the link and makes a real directory. It does not write into the target of the link, and it does not change the owner of that target. The `chown` of `/opt/spekk/cli` uses `-h`.
- When `~agent/.local/bin/spekk` is there and `~agent/.local` or `~agent/.local/bin` is a link, the install script does not change `spekk` in it. The deploy output says that it skipped that path, and why.
- `spekk sandbox status` runs `/opt/spekk/cli/spekk version` and `/opt/spekk/agent-client --version` as `agent`, not as the login user: with `runuser -u agent --` for a root login, and `sudo -n -u agent` for a login user that is not root.
- Live check: in a container with SSH, with `/opt/spekk/cli -> /etc/rootonly`, a deploy leaves `/etc/rootonly` unchanged (owner and contents) and installs the CLI into a real `/opt/spekk/cli/`. With `~agent/.local/bin` a link, the deploy says that it skipped it. `status` on a root login and on a sudo login prints both versions, and a shim shows that the version commands ran as `agent`.

**Tests:** `internal/sandbox/deploy_replace_test.go` (`TestInstallCommandDoesNotFollowLinksThatAgentOwns`). Live check (2026-10-18): two `ubuntu:24.04` containers with sshd and `--provider none`, one with a root login and one with `ubuntu` and sudo. With `/opt/spekk/cli -> /etc/rootonly`, the deploy from the earlier head changed `/etc/rootonly` to `agent:agent` and wrote the CLI into it; the deploy from this head left `/etc/rootonly` `root:root` with only its own file, and made a real `/opt/spekk/cli/`. With `~agent/.local/bin -> /etc/rootbin`, the earlier head replaced the root-owned `/etc/rootbin/spekk` with a link; this head kept it and printed `spekk: skipped /home/agent/.local/bin/spekk: a directory in that path is a link`. With a wrapper in place of each binary that logs `id -un`, `status` from the earlier head ran them as `root` and `ubuntu`; from this head it ran them as `agent` on both boxes, and printed both versions.
