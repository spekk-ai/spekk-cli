---
id: features-become-spec-files
parent: meeting-notes-to-specs
created: 2025-02-12T19:30:00Z
priority: 1
status: in_progress
---

# Features Become Proper Spec Files

Feature discussions from meetings are converted into proper spec files in the specs/ directory by the coach's meeting-processing skill.

## Success Criteria

- Feature discussions extracted from meeting → spec with assertions
- Each spec includes: id, priority, assertions with success criteria
- Specs follow existing format (no duplication of format rules from coach prompt):
  - YAML frontmatter with id, created (ISO 8601), priority, status
  - Kebab-case IDs
  - Clear success criteria for each assertion
- Coach proposes specs structure, waits for user approval before creating files
- Specs created in `specs/{spec-id}/` directory with assertion files in `assertions/` subdirectory
- Multiple features from one meeting → multiple separate specs (not combined)
