---
id: reconcile-writes-and-prunes
parent: install-reconciler
created: 2026-07-27T00:00:00Z
priority: 1
status: done
branch: feat/install-reconciler
depends-on: scan-owned-files
---

# Install Writes the Desired Set and Prunes the Rest

## Description

`spekk install` becomes a reconciler. It drives the managed files to the desired
final state for the target and scope. It writes the desired files with a stamp,
and it removes owned files that are not in the desired set.

## Success Criteria

- `Install` calculates the desired set: the agent shims and the `spekk-dev-loop`
  skill that the target and scope define now. It stamps each desired file with
  `StampContent` before it writes the file.
- `Install` writes or updates every file in the desired set.
- `Install` runs the scan and removes every owned file that is not in the desired
  set.
- `Install` returns a result that lists the written paths and the removed paths.
  The existing caller (`runInstallTargets`) still prints the written paths, and it
  also prints the removed paths.
- The reconcile is idempotent. A second run with the same target and scope writes
  the same content and removes nothing. A test proves this: run twice, and check
  that the second run reports no removals and leaves the same files.
- A test proves the prune: place a stamped file that is not in the desired set in
  a managed location, run `Install`, and check that the file is gone and appears
  in the removed list.
