---
id: sandbox-network-isolation
parent: sandbox-aws-cloudformation
created: 2026-08-29T21:00:00Z
priority: 1
status: not_started
---

# Sandbox instance lives in an isolated VPC with SSH-only ingress

The template creates its own VPC rather than using the default VPC. This keeps sandbox traffic isolated and makes teardown clean (`delete-stack` removes everything).

## Success Criteria

- Template creates a VPC with a single public subnet
- An Internet Gateway and route table provide outbound internet access
- A security group allows inbound TCP 22 (SSH) from a parameterized CIDR and all outbound traffic
- The SSH source CIDR parameter has no default — the user must explicitly set it (prevents accidental 0.0.0.0/0)
- `aws cloudformation delete-stack` removes all resources with no orphans
- No NACLs beyond the VPC default (keep it simple — SG is sufficient)
