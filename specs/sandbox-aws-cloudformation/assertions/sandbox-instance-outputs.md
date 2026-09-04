---
id: sandbox-instance-outputs
parent: sandbox-aws-cloudformation
created: 2026-08-29T21:00:00Z
priority: 1
status: not_started
---

# Template launches an Ubuntu instance and outputs a ready-to-use IP

## Success Criteria

- Template launches a single EC2 instance running Ubuntu 24.04 LTS (latest official Canonical AMI)
- AMI lookup uses `AWS::SSM::Parameter::Value<AWS::EC2::Image::Id>` referencing the Canonical SSM public parameter — no hardcoded AMI IDs
- Instance type is parameterized (default: `t3.medium`)
- Key pair name is a required parameter (no default — user must specify an existing key pair)
- Instance gets a public IP via subnet auto-assign (no Elastic IP needed)
- Root EBS volume is 50 GB gp3
- Stack outputs include: `PublicIP`, `InstanceId`, and a `SpekkCommand` output that prints the full `spekk sandbox create --provider manual --name <stack-name> --ip <ip> --ssh-key <key>` command ready to paste
- Instance has an `Name` tag of `spekk-sandbox-<stack-name>`
