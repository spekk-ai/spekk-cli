---
id: builder-loop-handles-parser-errors
parent: robust-error-handling
created: 2026-03-16T18:00:00Z
priority: 1
status: done
depends-on: parser-skips-specs-without-assertions-dir
---

# Builder Loop Handles Parser Errors Gracefully

**Tests:** `src/loops/__tests__/builder-loop-handles-parser-errors.test.js`

**Closes:** #7

## What Must Be True

The builder loop never crashes from a parser error. When `spekk next` fails or returns unexpected output, the builder logs the issue and retries after a delay — it does not `process.exit(1)`.

## Success Criteria

- Builder loop catches `spekk next` command failures (non-zero exit) and retries after a delay instead of exiting
- Builder loop handles malformed JSON from parser by logging a warning and retrying
- When all assertions are complete, builder logs a success message and waits for new work (already works — this assertion ensures parser errors don't prevent reaching that code path)
- Builder loop remains running through transient parser errors caused by specs being authored mid-flight
