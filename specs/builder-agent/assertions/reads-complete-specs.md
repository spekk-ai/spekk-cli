---
id: reads-complete-specs
parent: builder-agent
created: 2026-01-21T18:45:00Z
priority: 1
status: done
---

# Builder Must Read Complete Specification Files

## What Must Be True

**Builder agents must read complete specification files** - never partial content that could miss critical requirements.

When implementing an assertion, the builder must:

1. **Read the FULL assertion file** - every line, no truncation
2. **Read the FULL parent spec file** - complete context and requirements  
3. **Read related assertion files** - understand interconnected requirements
4. **Never rely on partial reads** - no line limits, no truncation

## Why This Matters

**Partial reads cause implementation errors:**
- Missing language/technology requirements (Node.js vs Python)
- Missing architectural constraints (standalone vs Django-integrated) 
- Missing explicit prohibitions ("Must NOT exist")
- Incomplete understanding of success criteria

**Recent example:**
- Spec required standalone Node.js parser in `app/parser/`
- Builder created Django management command instead
- Likely caused by reading only ~90 lines, missing Node.js requirement

## Implementation Requirements

**File reading behavior:**
- Use Read tool without line limits
- Read complete files: `Read(file_path, limit=None, offset=None)`
- For long files, read in chunks but process complete content
- Never truncate frontmatter or success criteria sections

**Required reading sequence:**
1. Read assigned assertion file (complete)
2. Read parent spec file (complete) 
3. Read related assertions if referenced
4. Validate understanding against ALL requirements

## Success Criteria

This assertion is "done" when:

- ✅ Builder reads complete assertion files (no line limits)
- ✅ Builder reads complete parent spec files  
- ✅ Builder demonstrates understanding of full requirements
- ✅ Builder catches technology/language requirements (Node.js vs Python)
- ✅ Builder respects architectural constraints (standalone vs integrated)
- ✅ Builder follows all explicit prohibitions ("Must NOT exist")
- ✅ No more implementation errors from partial reading
- ✅ Builder can explain WHY it chose specific implementation approach

## Test Cases

**Testable scenarios:**
1. Spec with Node.js requirement → Builder uses Node.js (not Python)
2. Spec with standalone requirement → Builder avoids framework integration  
3. Spec with "Must NOT exist" section → Builder removes/avoids prohibited items
4. Long spec file (>100 lines) → Builder references content from entire file
5. Assertion referencing parent spec → Builder shows knowledge of parent context

**Tests:** src/builder/__tests__/reads-complete-specs.test.js

## Implementation Notes

This is a **meta-requirement** about how builders operate, not what they build. It ensures quality execution of the spec-driven system itself.

**Critical for system reliability** - partial understanding leads to off-spec implementations that require rework.