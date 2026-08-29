---
id: sandbox-aws-cloudformation
created: 2026-08-29T21:00:00Z
priority: 1
---

# Sandbox AWS CloudFormation

A single CloudFormation template that stands up everything needed to run a spekk sandbox on AWS. One `aws cloudformation create-stack` command — no manual VPC, subnet, or security group setup.

The template creates an isolated network, launches an Ubuntu 24.04 instance with SSH access, and outputs the public IP. The user feeds that IP into `spekk sandbox create --provider manual` to provision and deploy the agent.

## Design constraints

- **One template, one stack** — no nested stacks, no cross-stack references. A user should be able to `create-stack` with a single file and be done.
- **Parameterized with safe defaults** — instance type, SSH source CIDR, key pair name are all parameters. Defaults should be secure (SSH locked to caller's IP, not 0.0.0.0/0).
- **Teardown is `delete-stack`** — no orphaned resources. Everything the template creates is owned by the stack.
- **No spekk-specific provisioning** — the template creates infrastructure only. Provisioning (Docker, Node, agent, etc.) is handled by `spekk sandbox create --provider manual` over SSH after the stack is up.
