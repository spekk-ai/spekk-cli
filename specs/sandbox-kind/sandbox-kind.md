---
id: sandbox-kind
created: 2026-10-15T23:40:00Z
priority: 1
---

# One-Off and Persistent Sandboxes

`spekk sandbox` does not know if a sandbox is for one task or for a long-lived agent. The only signal is a `tmp-` name prefix, which is a convention and not a rule. A one-off sandbox that nobody destroys continues to bill. Also, one machine cannot see the sandboxes that a different machine created. Issue #265 is the spec of record.

The local registry (`sandboxes.json`) stays the source of truth. Registration with spekk-app stays optional, so a user who has only the CLI gets all of this.

## Design

- **Registry fields.** `SandboxMeta` gets four optional fields: `kind` (`one-off` or `persistent`), `purpose`, `expiresAt` (RFC 3339) and `createdBy` (the hostname). An empty `kind` means `persistent`, so a record from an older binary does not change.
- **Create flags.** `--one-off` sets the kind to `one-off` with an expiry of 48 hours. `--expires <duration|date>` changes the expiry. `--purpose <text>` records why the sandbox exists. The default stays `persistent`.
- **Provider labels are optional.** A provider that can label its machines implements a separate `Labeler` interface. `Create` finds it with a type assertion. `Provider` does not change, so a provider that cannot label (a machine you already have, and the AWS CloudFormation path) needs no code. Label values obey the strictest rules, which are GCP's: lowercase `[a-z0-9_-]`, 63 characters or fewer.
- **Marker on the machine.** Credential injection also writes `/opt/spekk/sandbox.json` with the four fields. This works on every provider. The file holds no secret.
- **List.** `list` shows a kind column and flags an expired sandbox. `--expired` shows only the expired ones. `--all-hosts` also asks each labeling provider for the sandboxes that this machine did not create.
- **No auto-destroy.** Expiry is only a reminder.

## Rejected

- **A new method on `Provider`.** Each provider would need a stub, and a stub that does nothing hides which providers really label.
- **Labels in the create request.** A tag that the cloud refuses would then fail the create. A separate call after the machine exists makes a labeling failure a warning.
- **Auto-destroy at expiry.** A destroy is not recoverable, and a wrong clock or a wrong flag would delete real work.
- **An expiry on a persistent sandbox.** `--expires` without `--one-off` is an error. A persistent sandbox with an expiry has two meanings, and `list` could not show one clear kind.
