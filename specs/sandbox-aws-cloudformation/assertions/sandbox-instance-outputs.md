---
id: sandbox-instance-outputs
parent: sandbox-aws-cloudformation
created: 2026-08-29T21:00:00Z
priority: 1
status: not_started
---

# Template launches a prepared Ubuntu instance and outputs a ready-to-use IP

The instance has to reach the state a droplet reaches, because spekk checks for `/opt/spekk/.provisioned` and then injects credentials and deploys the agent without any preparation of its own. Cloud-init on EC2 reads the instance's UserData the same way it reads a droplet's user data, so the template's UserData carries the cloud-init content from `internal/sandbox/cloud-init.yaml`.

That content has one placeholder, the public key line for the `agent` user. On a droplet spekk fills it with a key it generated. On AWS the template fills it with a parameter, and a test proves that the parameter is the only difference between the two.

An AWS Ubuntu AMI disables root over SSH and gives the `ubuntu` user passwordless sudo, so the register command names `--ssh-user ubuntu`.

## Success Criteria

- Template launches a single EC2 instance running Ubuntu 24.04 LTS from the latest official Canonical AMI
- The AMI parameter has the type `AWS::SSM::Parameter::Value<AWS::EC2::Image::Id>` and defaults to the Canonical public SSM parameter, so no AMI ID is hard-coded
- Instance type is a parameter with the default `t3.medium`
- Key pair name is a required parameter with no default: the operator names an existing key pair, and it is the key the `ubuntu` login user accepts
- The public key for the `agent` user is a required parameter, `AgentPublicKey`, and it fills the placeholder line of `cloud-init.yaml`
- The UserData is the content of `internal/sandbox/cloud-init.yaml` with the placeholder line mapped to `AgentPublicKey`, and nothing else differs. A Go test compares the two and fails when they drift, and it also fails when `cloud-init.yaml` gains a `${` that `Fn::Sub` would read as a variable
- The UserData ends by writing `/opt/spekk/.provisioned`, because that is what `cloud-init.yaml` ends with
- Instance gets a public IP through subnet auto-assign, with no Elastic IP
- Root EBS volume is 50 GB gp3
- Stack outputs include `PublicIP`, `InstanceId`, and a `SpekkCommand` output that prints `spekk sandbox create --name <stack-name> --ip <ip> --ssh-key ~/.ssh/<key-name>.pem --ssh-user ubuntu`, ready to paste
- The stack name is the sandbox name, so the operator gives the stack a name that is also a valid spekk sandbox name: lowercase letters, digits, and hyphens
- Instance has a `Name` tag of `spekk-sandbox-<stack-name>`
