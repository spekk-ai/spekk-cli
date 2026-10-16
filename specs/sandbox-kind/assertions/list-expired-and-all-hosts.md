---
id: list-expired-and-all-hosts
parent: sandbox-kind
created: 2026-10-15T23:40:00Z
priority: 1
status: not_started
depends-on: one-off-create-labels-and-marker
branch: sandbox-kind
---

# `list` Shows the Kind, `--expired` Filters, and `--all-hosts` Finds Remote Sandboxes

**Tests:** internal/sandbox/kind_test.go

## Success Criteria

- `spekk sandbox list` has a `Kind` column and an `Expires` column. It sorts the rows by name. A sandbox whose expiry is in the past shows `EXPIRED` in the `Expires` column.
- `list --expired` shows only the expired sandboxes. With no expired sandbox, it says so.
- `Labeler` has a method that lists the labeled machines of that provider. `DOProvider` lists the droplets with the `spekk-sandbox` tag. `GCPProvider` lists the VMs in the default gcloud project that have a `spekk-kind` label.
- `list --all-hosts` asks each provider in `ValidProviders` that implements `Labeler`. It adds the machines that no local record names, with the status `remote`. It reads the kind and the expiry from the labels.
- For each cloud provider that `--all-hosts` cannot ask (the provider cannot be built, or its list call fails), `list` prints one line that names the provider and the reason, and it does not fail. A provider that does not label its machines (`none`) is not a cloud, so it gets no line.
- `--expired` and `--all-hosts` work together: the remote rows are filtered too.

## Verification

- Tests use fake providers through a seam. No test calls a real cloud API.
- A test checks the `--expired` filter, the `EXPIRED` mark, the remote rows, the removal of a remote machine that a local record already names, and the line for a provider that cannot be asked.
- `DOProvider` and `GCPProvider` list methods have tests against an `httptest` server and the `runGcloud` seam.
