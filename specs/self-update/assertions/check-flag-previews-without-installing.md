---
id: check-flag-previews-without-installing
parent: self-update
created: 2026-06-12T00:00:00Z
priority: 2
status: done
---

# `spekk update --check` Previews Without Installing

## Description

Users can find out whether an update is available without changing anything
on disk. `--check` performs only the API query — no asset download, no file
writes.

## Success Criteria

- `spekk update --check` queries the Releases API and compares versions
- When a newer version exists, it prints the current version, the latest
  version, and the hint to run `spekk update` to install
- When already up to date, it prints "Already on latest version (<version>)"
- In check mode the binary on disk is never modified and no release asset is
  downloaded
- Exit code is 0 in both outcomes (an available update is not an error)
