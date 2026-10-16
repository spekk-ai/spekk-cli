---
id: show-command
created: 2026-10-15T00:00:00Z
priority: 1
---

# `spekk show` Basics

## Overview

`spekk show` writes the Spec Explorer to `.spekk/index.html` and opens it. With `--watch` it starts a local server. This spec covers two basic behaviors that the other show specs (`show-cross-branch`, `show-mobile-cards`) do not cover: the help flag, a stable order for the specs and assertions in the output, and a `--watch` reload that does not depend on the file mtime.

## Assertions

- `show-help-exits-zero`: `spekk show --help` prints help and exits 0. It does not write the explorer or start a server.
- `show-order-matches-list`: the explorer data lists specs and assertions in the same order as `spekk list`, and two runs over the same specs give identical output.
- `watch-sees-same-tick-edit`: `spekk show --watch` reloads when a spec file gets new content, also when its mtime does not change.
