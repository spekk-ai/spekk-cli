---
id: atomic-binary-replace
parent: self-update
created: 2026-06-12T00:00:00Z
priority: 1
status: done
---

# The Running Binary Is Replaced Atomically

## Description

The update never leaves the user with a half-written `spekk` binary. The new
binary is staged as a temp file next to the target and swapped in with a
rename.

## Success Criteria

- The target path is the real location of the running binary:
  `os.Executable()` resolved through symlinks with `filepath.EvalSymlinks`
- The downloaded binary is written to a temp file (`.spekk-update-*`) in the
  same directory as the target, so the final `os.Rename` is atomic
  (no cross-filesystem move)
- On non-Windows platforms the temp file is chmodded `0755` before the
  rename, so the installed binary stays executable
- On Windows, where the running binary cannot be overwritten, the old binary
  is first renamed to `<path>.old`, then the temp file is renamed into place
- On any failure path the temp file is removed; the existing binary is left
  untouched if the swap never happened
- After a successful swap, "Updated successfully: <old> → <new>" is printed

**Tests:** `TestDownloadAndReplace` in `internal/update/update_test.go`
(verifies content replacement and the executable bit).
