---
id: one-off-create-labels-and-marker
parent: sandbox-kind
created: 2026-10-15T23:40:00Z
priority: 1
status: done
depends-on: registry-fields-default-persistent
branch: sandbox-kind
---

# `create --one-off --purpose x` Writes the Fields, the Labels and the Marker

A one-off sandbox carries its kind in three places: the local record, the cloud labels (DigitalOcean and GCP), and a marker file on the machine.

**Tests:** internal/sandbox/kind_test.go

## Success Criteria

- `spekk sandbox create` takes `--one-off` (a boolean), `--expires <duration|date>` and `--purpose <text>`. `CreateOptions` carries them as `Kind`, `ExpiresAt` and `Purpose`.
- `--one-off` sets the expiry to 48 hours after the create. `--expires` accepts a Go duration (`36h`), a number of days (`3d`), a date (`2026-10-17`, which means 00:00 UTC on that day) or an RFC 3339 time. A value that does not parse, that is too large to add to the current time, or that is not in the future, stops the create before anything bills, with an error that names the cause.
- `--expires` without `--one-off` is an error. Without `--one-off`, the kind is `persistent` and the record has no expiry.
- `Create` writes `kind`, `purpose`, `expiresAt` (RFC 3339, UTC) and `createdBy` (the hostname) to the record before the provider creates the machine, so a record that a failed create leaves has them too.
- `internal/sandbox` defines an optional interface `Labeler`. `Provider` does not change. After the provider creates the machine, `Create` labels it when the provider implements `Labeler`. The labels are `spekk-kind`, `spekk-host` and, when there is an expiry, `spekk-expires` (the expiry date in UTC, rounded up to the next day, so a remote list never flags a sandbox early).
- Each label value is lowercase `[a-z0-9_-]`, 63 characters or fewer. Other characters become `-`.
- `DOProvider` implements `Labeler` with DigitalOcean tags `key:value`. It creates each tag with `POST /v2/tags` first. The API returns 201 for a tag that exists already, so any other error from that call is an error. `GCPProvider` implements it with `gcloud compute instances add-labels` and `key=value`.
- A labeling error prints a warning to stderr. The create continues and can succeed.
- Credential injection writes `/opt/spekk/sandbox.json` in the same SSH command. It holds `kind`, `purpose`, `expiresAt` and `createdBy`, and no secret. The kind in the file is never empty: a record with no kind writes `persistent`.
- The marker write goes to a temporary file in `/opt/spekk` and then `mv -f` replaces the marker, so the write does not follow a symlink at the marker path.
- A failed marker write prints a warning on stderr. The credential step does not fail because of the marker.

## Verification

- A test runs `Create` with a fake provider that implements `Labeler`, and finds the four fields in the record and the three labels on the fake.
- A test runs `Create` with a fake labeler that returns an error, and finds a warning and no create error from the labeling.
- The real-cloud steps in the pull request prove the DigitalOcean tag requests and the GCP `add-labels` command. No unit test repeats the request strings that `Label` builds.
- A test fails if `equipSandbox` does not send `markerContent(meta)` to `injectCredentials`.
- A walkthrough on a machine you already have finds the marker with the four fields, and a create that cannot write the marker prints a warning and continues.
