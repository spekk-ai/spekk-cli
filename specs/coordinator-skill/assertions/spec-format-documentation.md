---
id: spec-format-documentation
parent: coordinator-skill
created: 2026-02-25T00:00:00Z
priority: 3
status: done
---

# Spec Format Documentation Updated with New Fields

Documentation specs that define YAML frontmatter format are updated to include `depends-on` and `branch` fields.

## Success Criteria

### Coach Agent Prompt

- [ ] Format validation section documents `depends-on` field (optional)
- [ ] Format validation section documents `branch` field (optional)
- [ ] Field descriptions explain purpose and format
- [ ] Examples show correct YAML structure

### Nested Spec Organization

- [ ] Group spec metadata list includes standard fields
- [ ] Assertion spec metadata list includes optional `depends-on` and `branch`
- [ ] Validation rules mention dependency ID validation
- [ ] Validation rules mention branch name validation

### Builder Agent Prompt

- [ ] Dependency-aware building process documented
- [ ] Builder checks `depends-on` before starting work
- [ ] Blocked assertions are skipped when dependency incomplete

### README

- [ ] Optional fields section includes `depends-on` and `branch`
- [ ] Default behavior documented (omitted = no dependency, main branch)
- [ ] CLI behavior documented (`spekk next` respects dependencies and branches)

## Validation

- [ ] All documentation files updated consistently
- [ ] No conflicting information across docs
- [ ] Terminology consistent across all files
- [ ] Examples use correct format
