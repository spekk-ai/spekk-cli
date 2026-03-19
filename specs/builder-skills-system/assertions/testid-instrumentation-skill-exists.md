---
id: testid-instrumentation-skill-exists
parent: builder-skills-system
created: 2026-03-19T18:06:00Z
priority: 1
status: draft
depends-on: builder-prompt-references-skills
---

# Test ID instrumentation skill exists with naming conventions

A builder skill file exists at `specs/builder-skills-system/testid-instrumentation-skill.md` containing the patterns for adding `data-testid` attributes to React components.

## Success Criteria

- File exists at `specs/builder-skills-system/testid-instrumentation-skill.md`
- Has **Triggers** section: activates when assertion mentions data-testid, test IDs, component instrumentation, or e2e testability
- Has **Workflow** section covering:
  - Which elements need data-testid (interactive elements, UI library components, content containers)
  - Priority levels (required, recommended, optional)
  - How to add data-testid to existing components
- Has **Patterns** section containing:
  - Naming convention: `[component-context]-[element-purpose]-[element-type]`
  - Examples for forms, modals, tables, dynamic elements in loops
  - Exclusions (pure presentational, test files, third-party wrappers, SVG, fragments)
- Content migrated from the `validate-testids --fix` mode of the slash command
