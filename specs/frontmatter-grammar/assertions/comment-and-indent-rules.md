---
id: comment-and-indent-rules
parent: frontmatter-grammar
created: 2026-08-28T22:00:00Z
priority: 2
branch: fix/frontmatter-scanner-grammar
status: done
---

# A Comment Is Invisible, and a Nested Line Sets Nothing

The scanner used to read a trailing `# ...` as part of the value, and it used to let an indented line set a top-level key. Both turned a legible file into a wrong value, and both were silent.

## Success criteria

- A trailing `#` that follows whitespace opens a comment, and the rest of the line is cut before any rule reads it. This holds on a key line and on a block-list item alike.
- Cutting a comment never destroys a list. A comment on the key that opens the list (`tags: # my tags`) leaves the items intact, and an item that holds only a comment (`- # placeholder`) is an empty item rather than the end of the list. Reading either as a plain line discards every item under the key.
- A `#` with no whitespace before it is data, so `link: https://example.com/x#frag` keeps its fragment.
- A quote opens a quoted scalar only where a value or an item starts. `note: "a # b"` keeps its hash; `apos: it's fine # note` loses its comment, because the apostrophe is a character in plain text. Inside a quoted scalar, `\"` and `''` do not close it.
- Top level is the shallowest column among the lines the scanner keeps, not column zero. A root mapping that is indented together parses, and the base is measured after comments and blanks are dropped — a comment shallower than the keys must not set the base below them.
- A deeper line is a nested child only when a key above it opened a region: one whose value is empty, or a block scalar (`key: |`). Such a line sets nothing — no known key, no custom field. Depth alone must not be enough: a top-level key written with one stray leading space has no parent, and swallowing it made a file that says `status: done` report `not_started` with no word about the indentation.
- A nested key closes an open block list only before that list has an item. Its own children belong to it, so `env:` / `matrix:` / `- linux` must not index linux as a value of `env`. After the list has an item, a deeper line is that item's own continuation (`- name: build` / `run: make`), and closing the list there dropped every later item of the list.
- A block-scalar header is `|` or `>`, and it may carry an indentation digit and a chomping sign in either order: `|`, `>-`, `|2`, `|+1`, `>2-`. Matching only the six unadorned spellings left every other header's body reading as top-level keys, so a `priority:` written in prose overwrote the real one.

**Known limits.** Three shapes read differently from a real YAML parser, all of them accepted rather than fixed. Once a block list has its first item, a deeper `- ` line joins that same top-level list rather than the nested key above it, so `env:` / `- first` / `  matrix:` / `    - linux` gives `env` both items; the rule above therefore holds only before the list's first item. A colon-less line at the base column ends an open list but not the region, so a later deeper key is still read as a child. And a stray-indented key with an empty value opens a region of its own. The last two are invalid YAML, which a lenient line scanner reads anyway; the first is unchanged from the behavior this spec replaces.

**Note:** the unclosed quote is a known limit, not an oversight, and it is a class rather than one case. A quote at any opener position — the start of a value or an item, or after a space, comma, colon, or bracket — opens a scalar that never closes when the line has no matching quote, and the comment then survives in the value: `note: 'tis fine # note`, `note: say "hello # world`, and `note: re-'do it # test` all keep their tail, while the near-identical `note: it's fine # note` loses it. A line scanner cannot separate an opening quote from a stray one without parsing the value. The next rule this scanner cannot express cleanly is the signal to adopt a YAML library instead.

**Tests:** `internal/parser/` — the comment rule including the list-opening and comment-only-item spellings and the two spellings that are data; an indented block with a shallower comment above it; a nested child and a nested list that set nothing; a list that survives a mapping item's continuation line; a stray indent that still reads as a top-level key, with and without a stray shallow line setting the base; a block-scalar header carrying an indentation indicator.
