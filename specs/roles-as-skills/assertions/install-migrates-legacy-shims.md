---
id: install-migrates-legacy-shims
parent: roles-as-skills
created: 2026-07-27T00:00:00Z
priority: 1
status: done
depends-on: roles-split-agents-and-skills
---

# Install Removes the Old Coach and Builder Agent Shims

## Description

The coach and builder are no longer agent shims. So the old agent shims must go
away when the user runs `spekk install`. The reconciler prunes a stamped shim.
The install also removes an unstamped legacy shim, because a user on an older
version has an unstamped shim that the reconciler does not own.

## Success Criteria

- After `spekk install` on a host that writes the coach and builder as skills, no
  `spekk-coach` or `spekk-builder` file is in the agent directory. The observer
  agent shim stays. (On a host with no skill path, the coach and builder stay as
  agent shims, and the install updates them in place.)
- A stamped coach or builder agent shim (one that a reconciler wrote) is pruned
  by the reconciler, because the desired set no longer contains it.
- An unstamped legacy coach or builder agent shim is also removed. The install
  checks the known legacy agent-shim paths. If such a file exists and its content
  is a spekk shim (it has the spekk shim signature, for example the instruction to
  run `spekk prompt`), the install makes a `<path>.bak` backup and removes the
  file. It records this in the result.
- The install does not remove a file at a legacy path that is not a spekk shim
  (a file the user wrote). It leaves that file.
- Tests cover a stamped legacy shim (pruned) and an unstamped legacy shim
  (backed up and removed). A live check in a temporary home confirms that a
  pre-existing `spekk-coach.md` agent shim is gone after `spekk install`.
