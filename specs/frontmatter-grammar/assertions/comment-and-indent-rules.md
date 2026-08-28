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
- A line deeper than that column sets nothing. It does not set a known key, it does not become a custom field, and it closes an open block list, so the items under a nested key never become values of the key above it.

**Note:** the apostrophe case is a known limit, not an oversight. `note: re-'do it # test` keeps its comment, because the quote follows a hyphen and reads as an opening quote. A line scanner cannot separate that from a real quoted scalar without parsing the value, and the next rule this scanner cannot express cleanly is the signal to adopt a YAML library instead.

**Tests:** `internal/parser/` — the comment rule including the list-opening and comment-only-item spellings and the two spellings that are data; an indented block with a shallower comment above it; a nested child and a nested list that set nothing.
