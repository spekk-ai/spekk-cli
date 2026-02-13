---
id: parent-status-synchronization
parent: spec-parser
created: 2026-01-22T20:00:00Z
priority: 1
status: done
---

# Parser Must Synchronize Parent Spec Status with Child Assertions

**Tests:** src/parser/__tests__/parent-status-sync.test.js

## What Must Be True

Parent spec files do NOT contain a `status` field in their YAML frontmatter. Parent status is always computed at runtime by the parser from child assertions. The parser must compute parent status during `parseAllSpecs()` and expose it in the returned data, regardless of whether the file contains a `status` field.

## Status Synchronization Rules

### Parent Status = "failed"
When **any** child assertion has status `failed`, the parent spec gets `failed` status.
- Takes priority over all other statuses
- Indicates blocking issues that must be resolved

### Parent Status = "done"  
When **all** active child assertions have status `done`, the parent spec gets `done` status.
- Draft assertions are ignored (not counted as active)
- Only applies when no children are failed

### Parent Status = "in_progress"
When **any** active child assertions have status `in_progress` or `not_started`, the parent spec gets `in_progress` status.
- Only applies when no children are failed
- Only applies when not all children are done

### Parent Status = "not_started"
When there are **no active child assertions**, the parent spec gets `not_started` status.
- Applies when spec has no assertions at all
- Applies when spec has only draft assertions

## Draft Assertion Handling

Draft assertions are excluded from parent status computation:
- Parent with children: `[done, draft, done]` → Parent status: `done`
- Parent with children: `[draft, draft]` → Parent status: `not_started`
- Draft assertions don't affect parent status in any way

## Priority Order

Parent status computation follows this priority:
1. **failed** (any child failed → parent failed)
2. **done** (all active children done → parent done)  
3. **in_progress** (any active child incomplete → parent in_progress)
4. **not_started** (no active children → parent not_started)

## Implementation Details

The parser must:
- Compute parent status automatically during `parseAllSpecs()`
- Apply synchronization after all assertions are loaded
- Use the `computeParentStatus()` function for consistency
- If a parent spec file contains a `status` field (legacy), ignore/override it with the computed value

## Success Criteria

- ✅ Parent spec files do not contain a `status` field (coach agent enforces this)
- ✅ Parent status automatically computed from child assertion statuses
- ✅ Failed children force parent to failed status (highest priority)
- ✅ All done children make parent done (when no failures)
- ✅ Any incomplete children make parent in_progress
- ✅ No active children make parent not_started
- ✅ Draft children are excluded from status computation
- ✅ Legacy `status` fields in parent files are overridden by computed status
- ✅ Status computation happens automatically during parsing