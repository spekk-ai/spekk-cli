---
id: comment-anchor-ids-exist
parent: comment-anchors
created: 2026-11-02T21:30:00Z
priority: 1
status: not_started
branch: feat/v2-comment-anchors
---

# Every Comment Anchor Names a Property That Exists

## Description

`spekk validate` scans the source files of the project for comment anchors. Each anchor must name an assertion ID in `specs/`. Scope of the scan and the syntax are in the parent spec.

## Success criteria

- A line that ends with `# spekk: <id>`, `// spekk: <id>` or `-- spekk: <id>` is an anchor for `<id>`. There can be whitespace between the comment marker and `spekk:`, there is at least one space between `spekk:` and the ID, and only whitespace follows the ID. The comment can follow code on the same line.
- A comment with more than one word after `spekk:` (`# spekk: the binary is installed`) is not an anchor. `spekk:managed` with no space is not an anchor.
- For each anchor whose ID is not an assertion ID, `spekk validate` prints one failure line `<path>:<line>: anchor names unknown property <id>` and exits 1. The failure lines sort with the other failures, as `validate-command` says.
- An anchor whose ID is not kebab-case (`# spekk: Balance`) is a failure of the same form. It is not ignored.
- The spec tree and the source of this repository at `main` 299283b have no anchor, so they validate with exit 0.
- Anchors in `.md` files, in a `testdata` directory, in binary files and in files that git ignores are not read.
- Only regular files are read. Symlinks, directory paths and gitlinks are skipped, including links to files outside the work tree and to devices.
- If git is not installed, no files are scanned, and `spekk validate` warns and exits 0 for an otherwise valid spec tree.
- A tree whose anchors all name existing properties validates as before: the same summary line and exit 0.
- **Tests:** `internal/validate/comment_anchors_test.go` checks unknown and known IDs, all three comment markers, prose, and skipped files.
