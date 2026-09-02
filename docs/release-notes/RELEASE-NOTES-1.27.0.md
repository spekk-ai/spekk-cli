# Spekk CLI 1.27.0 — A Sandbox That Does Not Admit Root

1.26.0 let a sandbox be a machine you already have. It assumed that machine lets you log in as root, and many do not: an AWS Ubuntu AMI gives you `ubuntu` and passwordless sudo, and disables root over SSH outright. This release adds the login user.

## `--ssh-user`

`spekk sandbox create --ip <address> --ssh-key <path> --ssh-user ubuntu` logs in as that user. The flag is for a machine you already have, so naming it is one more way to say there is nothing to create, and passing it with `--provider digitalocean` is an error: spekk reaches a droplet it made as root.

Four steps need root - inject the model credentials, configure the agent's git credentials, deploy the agent, tear it down - and each escalates with `sudo`, so the login user needs passwordless sudo. Everything else runs unprivileged. A sandbox recorded before this release has no login user, which reads as root, so nothing changes for one that already exists.

Two details are worth naming, because both are ways this could have gone wrong quietly.

Spekk stages the agent binary in the login user's home directory and lets root move it into `/opt/spekk` from there. The obvious place to stage it is `/tmp`, and that is the wrong place: a fixed name in a directory every local user can write is a file another local user can create first, and what root then moves into place is what systemd runs as the agent, next to an env file holding AWS keys and a GitHub token. On a droplet spekk made, nobody else has an account. This feature is for machines where somebody might.

The login user is checked before it is recorded, against a login name. Spekk puts the value in an ssh argument, and ssh reads an argument that starts with `-` as an option rather than as part of the destination, so `--ssh-user '-oProxyCommand=...'` would run a command on your own machine. The value is stored, so it would run again on every later `status`, `ssh`, `deploy` and `destroy`. A create that names an invalid user records nothing.
