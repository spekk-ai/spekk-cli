---
id: destroy-removes-marker
parent: sandbox-kind
created: 2026-10-16T01:10:00Z
priority: 2
status: not_started
depends-on: existing-machine-gets-fields-not-labels
branch: sandbox-kind
---

# `destroy` on a Machine You Already Have Removes the Marker

After `destroy`, the machine is not a sandbox. A marker that stays on it tells the next person who logs in the wrong thing. William decided this on 2026-10-16.

## Success Criteria

- The teardown that `destroy` runs on a machine you already have also removes `/opt/spekk/sandbox.json` with `rm -f`.
- The marker holds no secret, so a marker that cannot be removed does not make the teardown fail.
- `docs/cli-reference.md` and `docs/advanced/sandbox-architecture.md` say that `destroy` removes the marker.

## Verification

- A walkthrough creates a sandbox on a machine you already have, runs `destroy`, and finds no `/opt/spekk/sandbox.json` on the machine.
