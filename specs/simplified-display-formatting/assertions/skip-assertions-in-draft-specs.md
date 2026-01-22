---
id: skip-assertions-in-draft-specs
parent: simplified-display-formatting
created: 2026-01-22T23:10:00Z
priority: 1
status: not_started
---

# Skip Assertions in Draft Specs

The parser currently processes assertions from specs marked as `status: draft`, but should skip them entirely.

## What Must Be True

Parser excludes all assertions from any spec with `status: draft` from work consideration.

### Current Problem

1. `parseAllSpecs()` reads ALL assertions regardless of parent spec status
2. `findNextAssertion()` only filters individual assertions with `status: draft` 
3. Assertions from draft specs can still be picked up for work

### Required Changes

**In `findNextAssertion()` function:**
- Before filtering assertions, also exclude assertions whose parent spec has `status: draft`
- Check parent spec status in addition to assertion status
- Filter logic: `!['done', 'draft'].includes(a.status) && parentSpec.status !== 'draft'`

### Logic Update

```javascript
function findNextAssertion(assertions, specs) {
  // Filter to incomplete items, excluding:
  // - Assertions with status 'done' or 'draft'  
  // - Assertions whose parent spec has status 'draft'
  const incomplete = assertions.filter(a => {
    if (['done', 'draft'].includes(a.status)) return false;
    
    const parentSpec = specs.find(s => s.id === a.parent);
    if (parentSpec?.status === 'draft') return false;
    
    return true;
  });
  // ... rest of function
}
```

### Function Signature Update

- Update `findNextAssertion()` to accept both `assertions` and `specs` parameters
- Update calls to pass both parameters: `findNextAssertion(assertions, specs)`

## Success Criteria

- ✅ Assertions from draft specs are never returned by `findNextAssertion()`
- ✅ `spekk builder` ignores assertions in draft specs  
- ✅ Draft specs serve as true "planning/placeholder" status
- ✅ Function signature updated to accept specs parameter