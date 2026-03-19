---
id: api-audit-skill-exists
parent: builder-skills-system
created: 2026-03-19T18:07:00Z
priority: 1
status: not_started
depends-on: builder-prompt-references-skills
branch: feature/code-quality-qa
---

# API audit skill exists with route analysis patterns

A builder skill file exists at `specs/builder-skills-system/api-audit-skill.md` containing the patterns for analyzing page/component API call trees.

## Success Criteria

- File exists at `specs/builder-skills-system/api-audit-skill.md`
- Has **Triggers** section: activates when assertion involves building e2e tests, API mocking, or understanding a page's API surface
- Has **Workflow** section covering:
  - Analyzing router configuration (finding route files, parsing route hierarchy)
  - Identifying auth guards, layout wrappers, and their API calls
  - Tracing component import trees to find all API calls
  - Extracting endpoint information (baseUri, HTTP method, dynamic params)
- Has **Patterns** section containing:
  - Common API call patterns to look for (useQuery, useMutation, api.*, *Queries.*, direct fetch/axios)
  - Route hierarchy analysis (auth guards → layouts → pages → subcomponents)
  - How to find service definitions in `client/src/services/`
  - Output format for discovered API calls
- Content migrated from the `api-audit` slash command
- This skill is primarily used as a sub-step within the e2e-testing-skill, but can be used independently when the builder needs to understand a page's API surface
