---
icon: lucide/cloud
---

# A Sandbox on AWS

One CloudFormation template creates everything a spekk sandbox needs on AWS: an isolated VPC, one Ubuntu 24.04 instance prepared to the same state a DigitalOcean droplet reaches, and the command that registers the instance with spekk. One `create-stack` builds it, and one `delete-stack` removes it.

The template is `internal/sandbox/aws-cloudformation.yaml` in the [spekk-cli repository](https://github.com/spekk-ai/spekk-cli). It lives beside `cloud-init.yaml` because its UserData is a copy of that file, and a test in the repository fails when the two drift.

---

## What the stack creates

| Resource | Purpose |
|----------|---------|
| VPC, public subnet, internet gateway, route table | An isolated network with outbound internet access, which cloud-init needs to install packages |
| Security group | Inbound TCP 22 from the CIDR block you give, and all outbound traffic |
| EC2 instance | Ubuntu 24.04 from the latest Canonical AMI, `t3.medium` by default, with a 50 GB gp3 root volume and a `Name` tag of `spekk-sandbox-<stack-name>` |

The instance's UserData is the cloud-init content spekk gives a droplet. It installs Docker, Node.js, the GitHub CLI, and Claude Code, creates the `agent` user, enables UFW and fail2ban, and ends by writing `/opt/spekk/.provisioned`. That marker is what `spekk sandbox create` checks for before it injects credentials.

## Parameters

| Parameter | Default | Meaning |
|-----------|---------|---------|
| `KeyName` | none, required | An existing EC2 key pair. The `ubuntu` login user accepts it. |
| `SshCidr` | none, required | The IPv4 CIDR block that may reach port 22. Use your own address with `/32`. |
| `AgentPublicKey` | none, required | One OpenSSH public key line that the `agent` user accepts. Pass the public half of the key pair, so one private key reaches both users. |
| `InstanceType` | `t3.medium` | The EC2 instance type. |
| `UbuntuAmi` | Canonical's SSM parameter for Ubuntu 24.04 | Resolves to the latest AMI in the region. Keep the default. |

## Create the stack

Give the stack a name that is also a valid spekk sandbox name: lowercase letters, digits, and hyphens. The stack name becomes the sandbox name.

```bash
curl -fsSLO https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/internal/sandbox/aws-cloudformation.yaml

aws cloudformation create-stack \
  --stack-name my-sandbox \
  --template-body file://aws-cloudformation.yaml \
  --parameters \
    ParameterKey=KeyName,ParameterValue=my-key \
    ParameterKey=SshCidr,ParameterValue="$(curl -fsS https://checkip.amazonaws.com)/32" \
    ParameterKey=AgentPublicKey,ParameterValue="$(ssh-keygen -y -f ~/.ssh/my-key.pem)"

aws cloudformation wait stack-create-complete --stack-name my-sandbox
```

`ssh-keygen -y` prints the public key of a private key file, so you do not need a separate `.pub` file for a key pair you downloaded from AWS.

The stack reports `CREATE_COMPLETE` when the instance is running, not when cloud-init has finished. Cloud-init upgrades every package and installs Docker and Node.js, which takes several minutes. Wait for it before you register the machine:

```bash
ssh -i ~/.ssh/my-key.pem ubuntu@<PublicIP> cloud-init status --wait
```

## Read the outputs

```bash
aws cloudformation describe-stacks \
  --stack-name my-sandbox \
  --query 'Stacks[0].Outputs' \
  --output table
```

| Output | Value |
|--------|-------|
| `PublicIP` | The instance's public IPv4 address |
| `InstanceId` | The EC2 instance ID |
| `SpekkCommand` | The register command, ready to paste |

## Register the machine

`SpekkCommand` prints this, with the stack's values filled in:

```bash
spekk sandbox create --name my-sandbox --ip <PublicIP> --ssh-key ~/.ssh/my-key.pem --ssh-user ubuntu
```

Edit the key path if the private key is elsewhere. `--ssh-user ubuntu` is required: an AWS Ubuntu AMI disables root over SSH, and the `ubuntu` user has passwordless sudo, which the privileged steps use.

Spekk then checks for `/opt/spekk/.provisioned`, injects the credentials for the [auth mode](../configuration.md) you chose, deploys the agent binary, and starts the service. If the check fails, cloud-init has not finished yet, or the security group does not admit your address. Wait, or fix the `SshCidr` parameter with `update-stack`, and run the command again.

## Tear down

Remove the sandbox record first, so spekk stops the agent and takes its credentials off the machine, then delete the stack:

```bash
spekk sandbox destroy my-sandbox
aws cloudformation delete-stack --stack-name my-sandbox
aws cloudformation wait stack-delete-complete --stack-name my-sandbox
```

The stack owns every resource it created, so the deletion leaves nothing behind. If you delete the stack without `spekk sandbox destroy`, the machine and its credentials are gone with it, and `spekk sandbox destroy my-sandbox --force` then removes the stale local record.
