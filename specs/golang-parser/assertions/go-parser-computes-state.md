---
id: go-parser-computes-state
parent: golang-parser
created: 2026-04-05T12:03:00Z
priority: 1
status: done
depends-on: go-parser-reads-specs
branch: feature/golang-migration
---

# Go parser computes derived state

The Go parser implements the same derived state computation as the Node parser: parent status rollup, next priority algorithm, branch filtering, and lock handling.

## Success Criteria

**Parent status computation:**
- If any active child is `failed` → parent is `failed`
- If all active children are `done` → parent is `done`
- If any child is `in_progress` or `not_started` → parent is `in_progress`
- No active children → parent is `not_started`
- Draft children excluded from computation
- Parent specs with `status: draft` keep their draft status (not overwritten)

**Next priority algorithm:**
- Excludes assertions with status `done` or `draft`
- Excludes assertions whose parent spec is `draft`
- Filters by current git branch (unless `--all-branches`)
- Filters by `--spec` flag if provided
- Excludes assertions with unsatisfied `depends-on` (dependency not `done`)
- Excludes `in_progress` assertions with fresh (non-stale) locks
- Lock is stale if >2 hours old (parsed from `builder-{host}-{pid}-{timestamp}` format)
- Sorts by: priority (ascending) → created date (ascending) → id (ascending)
- Returns first match

**CLI flags:**
- `--all` returns full hierarchy JSON
- `--spec <id>` filters to specific spec
- `--assertion <id>` returns specific assertion directly
- `--all-branches` disables branch filtering
