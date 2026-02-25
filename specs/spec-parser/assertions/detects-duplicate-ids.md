---
id: detects-duplicate-ids
parent: spec-parser
created: 2026-01-20T16:28:00Z
priority: 2
status: done
depends-on: parses-frontmatter
branch: feature/spec-parser
---

# Parser Must Detect Duplicate IDs

**Tests:** src/parser/__tests__/spec-parser.test.js

## What Must Be True

The parser must detect and report duplicate `id` values across specs and within spec assertions.

## ID Uniqueness Rules

### Spec-Level IDs
- Each spec `id` must be unique across all specs
- No two spec files can have the same `id`
- Example: Only one spec can have `id: "spec-parser"`

### Assertion-Level IDs  
- Each assertion `id` must be unique within its parent spec
- Assertions in different specs CAN have the same `id`
- Example: Multiple specs can each have an assertion with `id: "validates-input"`

## Validation Scope

### Global Check (Specs)
```
specs/
├── spec-parser/spec-parser.md        (id: spec-parser)
└── living-dashboard/living-dashboard.md (id: spec-parser) ❌ DUPLICATE
```

### Local Check (Assertions within Spec)
```
specs/spec-parser/assertions/
├── validates-fields.md     (id: validates-fields)  
└── other-assertion.md      (id: validates-fields) ❌ DUPLICATE
```

## Error Messages

Parser should report clear duplicate ID errors:
- "Duplicate spec id 'spec-parser' found in files: specs/spec-parser/spec-parser.md, specs/other-spec/other-spec.md"
- "Duplicate assertion id 'validates-fields' in spec 'spec-parser' found in files: validates-fields.md, other-file.md"

## Success Criteria

- ✅ Parser detects duplicate spec IDs across all specs
- ✅ Parser detects duplicate assertion IDs within each spec
- ✅ Parser allows same assertion IDs in different specs
- ✅ Parser provides clear error messages showing conflicting files
- ✅ Parser reports ALL duplicates found (not just first one)