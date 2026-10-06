---
icon: lucide/cloud
---

# A Sandbox on Google Cloud

`spekk sandbox create --provider gcp` makes a Compute Engine VM, prepares it with the same cloud-init a DigitalOcean droplet gets, deploys the agent, and records the VM. `spekk sandbox destroy` deletes the VM and its disk. Spekk owns the machine from start to end, as it does a droplet.

The provider uses the `gcloud` CLI, so it uses the account, the project, and the zone that `gcloud` is configured with. Spekk does not keep a Google Cloud credential of its own.

## Before you start

- Install the [Google Cloud CLI](https://cloud.google.com/sdk/docs/install) and run `gcloud auth login`.
- Your account needs permission to create and delete VMs in the project, for example `roles/compute.instanceAdmin.v1`.
- The Compute Engine API must be on in the project: `gcloud services enable compute.googleapis.com`.
- The network must admit TCP 22 from your address. The `default` network has the `default-allow-ssh` rule, which does. For a different network, add a rule before you create the sandbox, because spekk does not create firewall rules.

## Create the sandbox

```bash
spekk sandbox create --provider gcp --name my-sandbox
```

| Flag | Meaning | Default |
|------|---------|---------|
| `--project` | The project ID | `gcloud config get-value project` |
| `--region` | The zone, for example `us-west1-b` | `gcloud config get-value compute/zone`, else `us-central1-a` |
| `--size` | The machine type | `e2-medium` (2 vCPU, 4 GB) |
| `--vpc` | The network name | the `default` network |

Spekk records the project and the zone it used. A later change to your `gcloud` configuration does not stop `destroy` from finding the VM.

## What spekk creates

One VM named `spekk-<name>`, from the latest Ubuntu 24.04 LTS image, with a 50 GB boot disk and an external IP address. Its metadata holds:

- `user-data`: spekk's cloud-init. It installs Docker, Node.js, the GitHub CLI, and Claude Code, creates the `agent` user, enables UFW and fail2ban, and ends by writing `/opt/spekk/.provisioned`.
- `ssh-keys`: a key that spekk generates, for a `spekk` login user. The Ubuntu image does not admit root over SSH, so the guest agent creates this user with passwordless sudo, and spekk logs in as it.
- `enable-oslogin=FALSE`: if the project turns OS Login on, the VM ignores `ssh-keys` and spekk cannot log in. This value also turns off IAM-based SSH access for this VM. An organization policy that requires OS Login overrides this value, and the provider does not work in that organization.
- `block-project-ssh-keys=TRUE`: the keys in the project's `ssh-keys` metadata do not reach the VM. The guest agent gives each of those users sudo, and with sudo a user can read the credentials that spekk injects. Use `spekk sandbox ssh` to log in.

The VM has no service account and no access scopes. The agent runs code from the repositories it clones, and a service account token from the metadata server would give that code access to the project.

If a VM with the name `spekk-<name>` already exists in the project, `create` stops and does not record it, because spekk did not make it. Choose another sandbox name.

## Tear down

```bash
spekk sandbox destroy my-sandbox
```

This deletes the VM and its boot disk, and removes the generated key and the local record. Nothing else stays in the project: the key is in the VM's metadata, and spekk created no other resource.
