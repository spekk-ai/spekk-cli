---
id: sandbox-aws-cloudformation
created: 2026-08-29T21:00:00Z
priority: 1
---

# Sandbox AWS CloudFormation

A single CloudFormation template that stands up everything a spekk sandbox needs on AWS. One `aws cloudformation create-stack` command, and no manual VPC, subnet, or security group setup.

The template creates an isolated network, launches an Ubuntu 24.04 instance with SSH access, prepares the instance to the state a droplet reaches, and outputs the public IP. The operator then registers the instance with `spekk sandbox create --name <stack-name> --ip <ip> --ssh-key <path> --ssh-user ubuntu`, and spekk injects credentials, deploys the agent, and starts it.

Spekk does not provision a machine it did not create (see `register-an-existing-machine`). It checks that `/opt/spekk/.provisioned` exists, and then it does only the part that is its own. So the preparation that cloud-init gives a droplet has to happen on the instance before spekk sees it, and the instance's UserData is where it happens.

## Design constraints

- **One template, one stack.** No nested stacks and no cross-stack references. An operator runs `create-stack` with one file and is done.
- **Parameterized with safe defaults.** Instance type, SSH source CIDR, and key pair name are parameters. A default is either safe or absent: the SSH source CIDR has no default, so nobody opens port 22 to `0.0.0.0/0` by accident.
- **Teardown is `delete-stack`.** Everything the template creates is owned by the stack, so the deletion of the stack leaves no orphaned resource.
- **UserData prepares the instance to the state of a droplet.** The UserData carries the same cloud-init content that `internal/sandbox/cloud-init.yaml` gives a droplet, and that content ends by writing `/opt/spekk/.provisioned`, the marker `spekk sandbox create` checks before it injects anything. The content in the template is not a third hand-maintained copy: `go generate` writes it from `cloud-init.yaml`, and a Go test compares the two and fails when they drift in any way other than the key parameter and the `default` user entry.
