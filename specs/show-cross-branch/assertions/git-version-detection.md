---
id: git-version-detection
parent: show-cross-branch
created: 2026-06-15T12:00:00Z
priority: 1
status: done
---

# Git Version Is Detected and Drives Graceful Degradation

## Description

Cross-branch mode relies on `git merge-tree --write-tree` (git ≥ 2.38) for honest
conflict detection. The installed git version must be detected so the feature
either uses full conflict detection or degrades gracefully on older git, never
silently producing wrong results.

## Success Criteria

- The installed git version is detected by shelling out to `git` via `os/exec`
  (e.g. `git --version`).
- When git ≥ 2.38, full conflict detection (via `merge-tree --write-tree`) is
  available and used.
- When git < 2.38, the feature degrades to **classification-only** mode:
  add / modify / delete are still classified from the three-way comparison, but
  conflicts are **not** confirmed via merge-tree.
- In degraded mode, the limitation is surfaced clearly to the user (e.g. a notice
  in the explorer and/or stderr) so conflict state is understood to be
  unavailable, not "none".
- Version parsing tolerates non-standard version strings (vendor suffixes, etc.)
  without crashing; if the version cannot be parsed, it degrades to
  classification-only rather than erroring.
