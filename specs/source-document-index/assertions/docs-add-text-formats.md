---
id: docs-add-text-formats
parent: source-document-index
created: 2026-10-17T16:00:00Z
priority: 1
status: done
branch: feature/source-document-index
depends-on: docs-search-command
---

# `spekk docs add` Files a Text Document Into raw/ and Writes Its Stub

## Success criteria

- `spekk docs add <path>...` takes files and folders. A folder is walked; files of an unsupported format are listed as skipped, and the walk continues.
- A file outside `raw/` is copied to `raw/<name>` (a folder to `raw/<folder>/...`). When the target exists with different content, that file fails with a message to rename it, and no file is overwritten. A file already under `raw/` stays where it is.
- The id is kebab-case of the path under `raw/` without the extension (`raw/Vendor Notes/API v2.md` -> `vendor-notes-api-v2`). The stub is `formatted/<id>.md`.
- The stub is frontmatter only, with keys in this order: `title`, `type`, `created` (today, `YYYY-MM-DD`), `source` (the path or URL given), `path`, `text` (only for a converted format), `sha256`, `size`, `tags`, `summary`. Strings are double-quoted. The title is the first heading, else the file name.
- `--summary "<text>"` sets the summary. It must have 40 to 600 characters after whitespace is collapsed, else that file fails and nothing is written for it. `--title` (one file only), `--type`, and `--tags a,b` set those fields.
- Without `--summary`, the stub has `summary: ""` and the command prints, for each such stub, the exact command to run after the agent reads the document: `spekk docs add <path> --summary "<1-3 sentences>"`.
- Adding a document again with unchanged content keeps the existing summary, title, type, and tags (a flag given on the command line replaces its field). With changed content, it updates `sha256`, `size`, and the derived text, and clears the summary unless `--summary` is given, and it says so.
- Formats in this assertion: `.md`, `.markdown`, `.txt` (indexed from the raw file), and `.html`, `.htm` (converted in the binary to `formatted/<id>.txt`: headings become `#` lines, `script` and `style` are dropped, block elements end a line, entities are decoded).
- After a successful add, the index has the document with kind `raw`, and `spekk docs search` finds it.
- Exit code 0 when every file succeeded, 1 when any failed. Each failure names its file.

**Tests:** the id rule, the stub round trip (write, then read back, with quotes and commas in the summary), the summary bounds, the re-add rules (unchanged keeps the summary, changed clears it), and HTML to text (headings, script, entities).
