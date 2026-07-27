---
id: hash-guard-backup
parent: install-reconciler
created: 2026-07-27T00:00:00Z
priority: 1
status: done
branch: feat/install-reconciler
depends-on: reconcile-writes-and-prunes
---

# The Reconciler Does Not Clobber a File the User Changed

## Description

A user can edit a managed file by hand. The reconciler must not overwrite or
remove such a file without a record. Before it writes over or removes a file, the
reconciler checks the body hash against the stamp. When the two do not agree, the
user changed the file.

## Success Criteria

- Before `Install` writes over a file that already exists, it reads the file. If
  the file has a stamp and the on-disk body hash does not agree with the stamp
  hash, `Install` makes a backup at `<path>.bak` and does not write over the file.
- Before `Install` removes an owned file, it applies the same check. If the hash
  does not agree, `Install` makes a `<path>.bak` backup and does not remove the
  file.
- `Install` records each such event as a warning in its result. The caller prints
  each warning to stderr.
- A pristine file (the hash agrees) is written over or removed with no backup and
  no warning.
- A file with no stamp at a desired path is checked by content. If the content
  is a spekk file (a role shim or the dev-loop skill), the reconciler owns it: it
  makes a `.bak` backup and updates the file to the current stamped content. This
  migrates a file that an older, pre-stamp version wrote. If the content is not a
  spekk file, the reconciler treats it as user content: it makes a `.bak` backup
  and does not write over it.
- Tests cover: a changed managed file at a desired path (backup, no overwrite), a
  changed managed file to be pruned (backup, no removal), and a pristine file
  (normal write or removal, no backup).
