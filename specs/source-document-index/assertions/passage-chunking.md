---
id: passage-chunking
parent: source-document-index
created: 2026-10-17T16:00:00Z
priority: 1
status: not_started
branch: feature/source-document-index
---

# A Document Is Cut Into Passages With Line Ranges

One function cuts the plaintext of any document into passages. The index, search, and show use the same line numbers.

## Success criteria

- A passage starts at each ATX heading (`#` to `######` followed by a space) and at each setext heading (a non-blank line followed by a line of only `=` or only `-`, three or more). A heading inside a fenced code block (```` ``` ```` or `~~~`) does not start a passage.
- A leading YAML frontmatter block (`---` on line 1 to the next `---`) is in no passage.
- Text before the first heading is one passage with an empty heading.
- A section that has no non-blank line after its heading is joined to the next section, so no passage is only a heading.
- A section of more than 80 lines is cut into windows of 40 lines. Each window keeps the heading of its section.
- Each passage has `start` and `end`: 1-based line numbers, inclusive, in the file as it is on disk (frontmatter lines counted). Blank lines at the end of a passage are not in its range.
- A file with no non-blank text gives no passages, and not an error.
- CRLF line endings give the same line numbers as LF.

**Tests:** table tests for heading kinds, code fences, frontmatter offset, heading-only merge, the 80/40 window rule, and CRLF. These are the chunk-boundary edge cases.
