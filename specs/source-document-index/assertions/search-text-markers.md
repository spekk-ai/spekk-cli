---
id: search-text-markers
parent: source-document-index
created: 2026-10-17T19:00:00Z
priority: 1
status: not_started
branch: feature/source-document-index
depends-on: hostile-documents-are-contained
---

# The Text Output of `spekk docs search` Puts Document Text Between Nonce Markers

`docs show` puts document text between begin and end markers that hold a value that is new for each call. The text output of `search` prints document lines and summaries with only a notice in its first line, so a document line can imitate the footer or a result.

## Success criteria

- When the results hold a document of kind `doc` or `raw`, the text output has one begin marker before the first result and one end marker after the last result. Both markers hold the same random value, which is new for each call, and use the words of the `docs show` markers.
- The footer and the "more results left out" line are after the end marker.
- The text output stays under the 4,000-byte cap.
- Output with only spec and assertion results has no markers.

**Tests:** `internal/index/search_test.go`, one assertion in `TestSearchCleansDocumentText`.
