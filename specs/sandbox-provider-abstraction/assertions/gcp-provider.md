---
id: gcp-provider
parent: sandbox-provider-abstraction
created: 2026-10-06T00:00:00Z
priority: 2
status: done
depends-on: cli-provider-dispatch
branch: feature/sandbox-gcp-provider
---

# Google Cloud Is a Provider Implementation

An operator who works in Google Cloud has no path to a sandbox today except to build a VM by hand and register it with `--provider none`. Spekk then does not own the machine, so `destroy` leaves it running. The GCP provider makes a Compute Engine VM the same thing a droplet is: spekk creates it, provisions it with the same cloud-init, and deletes it.

The provider drives the `gcloud` CLI and does not call the Compute API directly. An operator in GCP already has `gcloud` and its credentials, and `gcloud` resolves Application Default Credentials, OS Login, and service account impersonation for us. A direct API client needs an OAuth library, or a token that `gcloud` prints anyway. Spekk already shells out to `ssh` and `ssh-keygen` for the same reason.

The flags keep their names. `--region` takes a zone, because a VM lives in one zone and the metadata field that `list` shows is `Region`. `--project` takes the project ID, `--size` takes the machine type, and `--vpc` takes the network name. The temptations rejected here are a `--zone` flag with per-provider flag scoping, a firewall rule that spekk creates and must remember to delete, and a fallback to the Compute REST API when `gcloud` is absent.

An Ubuntu image on Compute Engine does not admit root over SSH. The provider puts the generated key in the instance's `ssh-keys` metadata for a `spekk` login user, and the guest agent creates that user with passwordless sudo. That is the non-root path that `non-root-login-user` built, used for a machine spekk did create. So the provisioning wait, which until now logged in as root because a droplet admits root, logs in as the recorded login user.

## Success Criteria

- `--provider gcp` selects a `GCPProvider` that implements `Provider`. `ProviderByName("gcp")` returns it, and returns an error that says to install the Google Cloud CLI when `gcloud` is not on `PATH`.
- `GCPProvider.Create` resolves the project from `--project`, else from `gcloud config get-value project`, and fails before it creates anything when neither gives one, or when the value is not in the form of a project ID, because gcloud refuses a project name before it asks the API, and the record of that failure could never be destroyed. It resolves the zone from `--region`, else from `gcloud config get-value compute/zone`, else `us-central1-a`. The machine type defaults to `e2-medium`, which matches the 2 vCPU and 4 GB of the droplet default. It records the resolved project, zone, and machine type in `Project`, `Region`, and `Size`, because `Destroy` must find the VM after the operator changes their `gcloud` configuration.
- `GCPProvider.Create` generates a key pair, renders cloud-init with the public key, and runs `gcloud compute instances create spekk-<name>` with Ubuntu 24.04 LTS from the `ubuntu-os-cloud` image family, a 50 GB boot disk, the cloud-init as `user-data` metadata, `ssh-keys` metadata for the `spekk` user, `enable-oslogin=FALSE`, so a project that turns OS Login on does not silently ignore the key, and `block-project-ssh-keys=TRUE`, because the guest agent gives every project key's user sudo, and sudo reads the injected credentials. The VM has no service account and no scopes, because the agent runs code from the repositories it clones. `--vpc` passes through as `--network`.
- The VM name goes in a new `gcpInstance` metadata field. The VM's public address goes in `IP`, and `SSHUser` is `spekk`. When the create command fails because the name is taken, the provider does not record the VM: spekk did not make it, and `destroy` would delete somebody else's VM. When it fails in another way, the provider asks `gcloud` whether the VM exists anyway, and keeps the name unless `gcloud` says it is not found or that the zone or the VM name is invalid, which no VM can have, because a VM with no metadata entry is one that `destroy` cannot reach.
- `GCPProvider.Destroy` deletes the VM with its boot disk, and leaves any other disk to its own auto-delete setting, and it treats a VM that is already gone as done. It refuses with `ErrNoMachineRecorded` when no VM name is recorded. No cloud-side key remains to delete, because the key is in the VM's metadata.
- `GCPProvider.Status` returns the VM status from `gcloud compute instances describe`, for example `RUNNING`, and an empty string when no VM name is recorded.
- `namesMachine` and `machineRef` know the VM name, so a failed create saves the record, and the destroy prompt names the VM.
- The provisioning wait logs in as `sshUser(meta)`. For a droplet that is root, as before.
- Every `gcloud` call goes through one seam, so the tests run without `gcloud` and without a cloud account.
- `docs/advanced/sandbox-gcp.md` tells an operator what the provider creates, the flags and their defaults, and that the network must admit TCP 22. The `create` help text lists `gcp`.
- A live check against a real project creates a sandbox, reports its status, and destroys it, and leaves no VM or disk behind.

**Tests:** internal/sandbox/gcp_provider_test.go
