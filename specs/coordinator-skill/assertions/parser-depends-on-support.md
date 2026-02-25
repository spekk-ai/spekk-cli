---
id: parser-depends-on-support
parent: coordinator-skill
created: 2026-02-25T00:00:00Z
priority: 2
status: done
---

# Parser Validates and Parses depends-on Field

Spec parser reads and validates the `depends-on` field in assertion frontmatter.

## What Must Be True

### Field Parsing
- [ ] Parser extracts `depends-on` field from YAML frontmatter
- [ ] Field name is converted to camelCase (`dependsOn`) in parsed object
- [ ] Omitted field is treated as null (no dependency)
- [ ] Explicit `null` value is accepted

### Field Validation
- [ ] `depends-on` must be a string or null if present
- [ ] String values must be kebab-case (lowercase with hyphens)
- [ ] Referenced assertion ID must exist in the spec tree
- [ ] Assertion cannot depend on itself
- [ ] No circular dependency chains are allowed

### Circular Dependency Detection
- [ ] Dependency chains are validated after all assertions are parsed
- [ ] Cycles are detected by walking the full dependency graph
- [ ] Circular dependency errors show the complete cycle path
- [ ] Error messages explain how to break the cycle

### Error Handling
- [ ] Type errors identify the file and invalid value
- [ ] Format errors specify expected kebab-case format
- [ ] Missing reference errors name the non-existent ID
- [ ] Self-reference errors are clear
- [ ] Circular dependency errors show full path
- [ ] All error messages are actionable

## Validation Points

- Parser successfully reads `depends-on` field
- Field name converted to `dependsOn`
- Invalid types are rejected
- Invalid formats are rejected
- Missing references are rejected
- Self-references are rejected
- Circular dependencies are detected
- Error messages guide correction
