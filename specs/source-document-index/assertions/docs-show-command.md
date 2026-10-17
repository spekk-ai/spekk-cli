---
id: docs-show-command
parent: source-document-index
created: 2026-10-17T16:00:00Z
priority: 1
status: not_started
branch: feature/source-document-index
depends-on: docs-search-command
---

# `spekk docs show` Prints Numbered Lines of One Document

## Success criteria

- `spekk docs show <id>` finds the document by id in the index. A path relative to the repository root (`docs/concepts.md`, `raw/x.pdf`) also works. When two kinds share the id, the command lists both and asks for `--kind`.
- It prints a header (`kind`, `id`, title, `text_path`, the line range shown, and the total line count), then the lines with their numbers (`  42  text`).
- `--lines a:b` prints lines a to b inclusive. `a:` prints from a. A range outside the file is cut to the file. A malformed range is an error with exit code 2.
- At most 200 lines are printed for each call. When the range is larger, the output says which lines it printed and gives the next `--lines` value.
- For kinds `raw` and `doc`, the lines are between a begin and an end marker that say the text is untrusted data. C0 control characters other than tab are removed.
- An unknown id exits 1 with `docs show: no document "<id>"`.

**Tests:** the 200-line cap and the next-range hint, and range parsing.
