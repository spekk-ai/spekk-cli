---
id: prompts-recommend-query-first
parent: agents-query-the-index
created: 2026-08-21T18:20:00Z
priority: 2
branch: fix/warning-discipline-and-lock-model
status: done
---

# The Coach and Builder Prompts Recommend `spekk query` First

Both prompts name `spekk query` as the way to find an assertion by keyword or by any metadata field, and neither pipes `spekk list` into `grep`.

## Success criteria

### `specs/coach-agent/coach.prompt.md`
- The line `spekk list --json | grep "keyword"` is **removed**. It returns one JSON line without the `id` or the `file`, so the match cannot be acted on.
- A `spekk query` example takes its place and shows a keyword search over titles, for example:
  ```bash
  spekk query "SELECT id, status, file FROM assertions WHERE title LIKE '%keyword%'"
  ```
- One short sentence says why: `query` reads `.spekk/index.db`, which the command refreshes first, and it returns whole rows.
- The `grep -rl "keyword" specs/` guidance for a search of spec prose **stays**, with a sentence that names it as the one search `query` cannot serve, because no table holds an assertion body.
- The `spekk list --status <value>` examples stay. Enumerating one status is a job `list` does well, and this assertion does not remove it.

### `specs/builder-agent/builder.prompt.md`
- The "Preferred Tool Patterns" section names `spekk query` alongside `spekk next` and `spekk list`, with one example that filters on a field `list` has no flag for, such as `branch`.
- `spekk next` stays the first recommendation for "what do I work on now". It answers that question in one call, and no query beats it.

### Both
- The examples run against the real schema: tables `specs(id, title, status, priority, branch, file)`, `assertions(id, parent_id, title, status, priority, branch, file)`, and `depends_on(assertion_id, depends_on_id)`. An example that names a column outside this set is a defect.
- No other behavior in either prompt is removed or contradicted.

**Note:** This is prompt prose only. Add no code and no CLI flag.

**Tests:** Prompt prose has no unit test. Verify by running every example in the changed sections against this repository and confirming each returns rows without an error, and by confirming `spekk prompt coach` and `spekk prompt builder` render the new text.
