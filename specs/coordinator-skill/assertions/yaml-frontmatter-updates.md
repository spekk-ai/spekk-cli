---
id: yaml-frontmatter-updates
parent: coordinator-skill
created: 2026-02-25T00:00:00Z
priority: 1
status: done
depends-on: branch-assignment
---

# Coordinator Updates YAML Frontmatter with Dependencies and Branches

Coordinator writes `depends-on` and `branch` fields to assertion YAML frontmatter and commits changes.

## Success Criteria

### YAML Frontmatter Must Contain

- [ ] `depends-on` field added only when dependency exists (omitted = no dependency)
- [ ] `branch` field added to all assertions
- [ ] New fields inserted in logical order (after status, before other custom fields)
- [ ] Existing YAML fields preserved without modification
- [ ] Existing markdown content below frontmatter unchanged
- [ ] YAML formatting preserved (indentation, spacing)

### Commit Requirements

- [ ] Single commit contains all YAML updates
- [ ] Commit message lists affected branches and assertion counts
- [ ] Commit message shows dependency chains
- [ ] No changes to spec content or existing metadata

### Validation (Parse Don't Validate)

- [ ] After YAML updates, run `parseAllSpecs()` to validate structure
- [ ] Parser catches invalid dependency IDs (non-existent assertions)
- [ ] Parser catches circular dependencies
- [ ] Parser catches invalid branch names
- [ ] Parser catches malformed YAML
- [ ] If parser reports errors, show them to user and abort
- [ ] No manual validation logic - let parser do its job

### User Confirmation

- [ ] User can preview changes before commit
- [ ] User can confirm or cancel before writing files
