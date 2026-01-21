---
id: identifies-next-priority
parent: spec-parser
created: 2026-01-20T16:15:00Z
priority: 2
status: done
---

# Parser Must Identify Next Priority Item

## What Must Be True

The parser must identify the single highest-priority incomplete assertion to work on next.

## Priority Algorithm

1. **Read all specs and assertions** from `specs/` directory
2. **Filter to incomplete items:**
   - Include items where `status != "done"`
   - This includes `not_started` and `in_progress`
3. **Work at assertion level:**
   - Return next incomplete assertion, not spec
   - Assertions are the atomic unit of work
4. **Sort by priority:**
   - Priority 1 items come first (highest)
   - Then priority 2 items
   - Then priority 3 items (lowest)
5. **Break ties (deterministic):**
   - If multiple items have same priority, select oldest by `created` timestamp
   - Older items worked on first (encourages finishing old work before adding new)
6. **Return single item:**
   - Output the one highest-priority incomplete assertion

## Special Cases

**All assertions done for a spec:**
- If a spec has all assertions `done`, mark the spec as `done` (status aggregation)
- Move to next spec's assertions

**All work complete:**
- If all assertions in all specs are `done`, output indicating no work remaining
- Exit with success status

**No specs exist:**
- If `specs/` directory is empty, output indicating no specs found
- Exit with success status

## Output Format

```json
{
  "type": "assertion",
  "id": "validates-required-fields",
  "parent": "spec-parser",
  "file": "specs/spec-parser/assertions/validates-required-fields.md",
  "priority": 1,
  "status": "not_started",
  "title": "Parser Must Validate Required Fields",
  "content": "... full markdown content ...",
  "spec": {
    "id": "spec-parser",
    "file": "specs/spec-parser/spec-parser.md",
    "title": "Spec Parser"
  }
}
```

## Success Criteria

- ✅ Parser returns highest priority incomplete assertion
- ✅ Parser never returns completed items
- ✅ Parser works at assertion level (not spec level)
- ✅ Parser handles ties correctly (by created date, then id)
- ✅ Parser handles "all done" case gracefully
- ✅ Parser handles "no specs" case gracefully

**Tests:** `tests/spec-parser.test.js`
