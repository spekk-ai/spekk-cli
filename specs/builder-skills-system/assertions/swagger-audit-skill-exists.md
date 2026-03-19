---
id: swagger-audit-skill-exists
parent: builder-skills-system
created: 2026-03-19T19:30:00Z
priority: 2
status: draft
depends-on: builder-prompt-references-skills
---

# Swagger audit skill exists with OpenAPI validation patterns

A builder skill file exists at `specs/builder-skills-system/swagger-audit-skill.md` (or project-local `.spekk/builder-skills/swagger-audit-skill.md`) containing the patterns for ensuring OpenAPI schema accurately reflects Django implementation.

## Success Criteria

- Has **Triggers** section: activates when assertion mentions OpenAPI, swagger, schema documentation, `@extend_schema`, or API documentation accuracy
- Has **Workflow** section covering:
  - Validation chain: migrations → models → serializers → viewsets → OpenAPI
  - Checking `@extend_schema` decorators exist on custom actions
  - Verifying serializer constraints match migration field definitions (max_length, null, choices)
  - Ensuring query parameters are documented (filterset_fields, search_fields)
- Has **Patterns** section containing:
  - Type mappings (Django/DRF field → OpenAPI type)
  - Common `@extend_schema` decorator patterns for actions
  - Serializer constraint enforcement patterns
  - Edge cases (abstract models, proxy models, polymorphic serializers)
- Content migrated from the `swagger-audit` slash command
- Project-local skill since it depends on Django/DRF and MCP OpenAPI tooling
