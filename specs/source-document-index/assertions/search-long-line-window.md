---
id: search-long-line-window
parent: source-document-index
created: 2026-10-17T23:30:00Z
priority: 1
status: done
branch: feature/transcript-search-lines
depends-on: docs-search-command
---

# Search Prints the Part of a Long Line That Matches

A transcript often has one line for each speaker turn, and a turn can have hundreds or thousands of characters. Search cuts each printed line to 160 characters, so it prints the start of the turn, and the answer later in the turn is not in the output. The agent then makes one more call (`docs show`) to read it, and each call costs the whole context again.

An eval on a private corpus of meeting notes and transcripts found the gold transcript at rank 1, but the key not in the search output, because the key was after character 160 of its line. Each extra model call cost about 10,000 tokens in every arm.

## Success criteria

- A printed line that holds a query word and has more than 160 characters is cut to the 160-character part with the most distinct query words. A tie goes to the part with more matches in total, then to the earlier part. A part starts at the start of the line, or at a word about 40 characters before a match, so the words that lead to the match are printed too.
- A part that does not start at the start of the line begins with `…`, and a part that does not end at the end of the line ends with `…`. A line of 160 characters or fewer is printed whole, as before.
- A line that holds no query word (a line next to a match) is cut to its first 160 characters, as before.
- The score of a line, the order in which lines are picked, and the budget rule do not change: a line costs the bytes of its printed text plus 8, as before. Only the text of a long matching line changes, so a window with two marks can cost a few bytes more than the cut start of the same line.
- The same lines are in the text and the JSON output (`lines[].text`).

**Tests:** `internal/docs/terms_test.go`: a long line with the query words near its end prints that part with the two marks; a tie between two parts goes to the one with more matches; a short line and a line with no match are unchanged.
