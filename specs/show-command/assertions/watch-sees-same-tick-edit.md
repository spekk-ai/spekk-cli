---
id: watch-sees-same-tick-edit
parent: show-command
created: 2026-10-16T00:00:00Z
priority: 1
status: done
---

# `spekk show --watch` Sees an Edit That Keeps the Same mtime

## Description

The `--watch` poller must reload when the content of a spec file changes, also when the file modification time (mtime) does not change. A filesystem records mtime with a coarse clock (for example one kernel tick, or one second on some filesystems). Two saves of the same file inside one tick get the same mtime. Before this fix, the poller compared only mtimes. When the poller read the file between the two saves, it did not see the second save, and the explorer kept the old content until the next save in a different tick.

**Tests:** internal/show/watch_test.go (`TestWatchSpecsDetectsChange`)

## Success Criteria

- When a `.md` file under `specs/` gets new content with the same mtime and the same size as before, the watcher calls its change callback on its next poll.
- A new `.md` file and a deleted `.md` file still cause a reload, as before.
- A poll where no `.md` file content changed does not cause a reload.
- Live check: with `spekk show --watch` running, save a spec two times within one mtime tick. The explorer shows the second save.
