---
id: sandbox-network-isolation
parent: sandbox-aws-cloudformation
created: 2026-08-29T21:00:00Z
priority: 1
status: done
branch: feature/sandbox-aws-cloudformation
---

# Sandbox instance lives in an isolated VPC with SSH-only ingress

The template creates its own VPC and does not use the default VPC. This keeps sandbox traffic isolated and makes teardown clean: `delete-stack` removes everything, because the stack owns everything.

The security group is the network-level firewall. The UserData also enables UFW on the host with the same rule, inbound port 22 only, because that is what a droplet gets. The two agree, and the instance defends itself even if somebody later widens the security group.

## Success Criteria

- Template creates a VPC with a single public subnet
- An Internet Gateway and a route table provide outbound internet access, which the UserData needs to install packages
- A security group allows inbound TCP 22 (SSH) from a parameterized CIDR and all outbound traffic
- The SSH source CIDR parameter has no default, so the operator must set it explicitly, which prevents an accidental `0.0.0.0/0`
- `aws cloudformation delete-stack` removes all resources with no orphans
- No NACLs beyond the VPC default. The security group is sufficient
