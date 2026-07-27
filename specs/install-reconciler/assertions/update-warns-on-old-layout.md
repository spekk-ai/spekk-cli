---
id: update-warns-on-old-layout
parent: install-reconciler
created: 2026-07-27T00:00:00Z
priority: 2
status: done
branch: feat/install-reconciler
depends-on: scan-owned-files
---

# Update Warns When an Old Layout Is Present

## Description

After a self-update, the installed files can be from an old layout. `spekk
update` checks the install locations and warns. It does not change any file,
because update owns the binary and install owns the harness files.

## Success Criteria

- A function calculates, for each supported target and each scope, the owned set
  (from the scan) and the desired set. It reports the owned files that are not in
  the desired set. This function reads files only. It writes and removes nothing.
- `spekk update` calls this function after the version check. If it finds any
  owned file that is not in the desired set, it gives a warning. The warning lists
  the stale files and shows the `spekk install --target <tool>` command that does
  the migration.
- If it finds no stale file, `spekk update` gives no such warning.
- `spekk update` changes no managed file in any case.
- A test drives the check function with a stamped file that is not in the desired
  set and confirms the function reports it. A second test with only desired files
  confirms the function reports nothing.
