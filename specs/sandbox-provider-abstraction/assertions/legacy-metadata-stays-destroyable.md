---
id: legacy-metadata-stays-destroyable
parent: sandbox-provider-abstraction
created: 2026-08-29T21:30:00Z
priority: 1
status: done
depends-on: provider-interface
branch: feat/provider-interface
---

# A Sandbox Created Before the Provider Field Stays Destroyable

Live sandboxes have metadata written by a binary that predates every part of this work. Their entries carry `dropletId` and `sshKeyId` and no `provider`. Introducing the abstraction must not strand them, because the failure mode is silent and expensive: teardown is skipped, the local record is deleted anyway, and a droplet bills forever with nothing left on disk to identify it.

## Success Criteria

- An entry with no `provider` field is read as DigitalOcean, which was the only provider when such entries were written. An entry naming an unknown provider is an error, not a default.
- `spekk sandbox destroy` on such an entry deletes the droplet and the DigitalOcean SSH key, then removes the local metadata — the same sequence it has always run.
- Saving or removing any entry leaves every other entry's fields intact on disk. A rewrite that drops another sandbox's droplet id is unrecoverable.
- `spekk sandbox status` shows the droplet id when there is one, and marks a status that came from the file rather than from the API, so a stale value cannot read as live.
- The destroy prompt names the machine it is about to destroy, not only the sandbox name.
- `--force` clears a record that names no machine, so an entry whose droplet was already removed by hand is not stuck forever. Without `--force` the refusal stands.

**Tests:** internal/sandbox/provider_test.go
