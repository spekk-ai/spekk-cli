---
id: scan-owned-files
parent: install-reconciler
created: 2026-07-27T00:00:00Z
priority: 1
status: done
branch: feat/install-reconciler
depends-on: managed-stamp
---

# A Scan Finds the Owned Files in the Known Locations

## Description

The reconciler must know which files it owns before it can prune. It learns this
from a scan, not from a manifest. The scan looks in the known managed locations
for a target and scope, and it keeps only the files that carry the stamp.

## Success Criteria

- A function scans the managed locations for a given target and scope
  (global or project) and returns the owned set. Each owned entry has the file
  path, the hash from the stamp, and a flag that says whether the on-disk body
  agrees with the stamp hash.
- The managed locations for a target and scope are the agent directory and the
  skill directory that the target descriptor already defines. The scan reads
  every file in those directories.
- The scan keeps only files whose content has the stamp. It ignores a file with
  no stamp (for example a file that the user wrote).
- The scan does not fail when a managed directory does not exist. It returns an
  empty set for that location.
- The function lives in `internal/install` and has unit tests, including a test
  with a stamped file, a user file with no stamp, and a missing directory.
