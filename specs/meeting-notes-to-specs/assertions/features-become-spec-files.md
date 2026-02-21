---
id: features-become-spec-files
parent: meeting-notes-to-specs
created: 2025-02-12T19:30:00Z
priority: 1
status: done
---

# Features Become Proper Spec Files

Feature discussions from meetings are converted into proper spec files in the specs/ directory.

**Tests:** src/meeting-processor/__tests__/cli.test.js

## Success Criteria

- Feature discussions extracted from meeting → spec with assertions
- Each spec includes: id, priority, assertions with success criteria
- Specs follow existing format:
  - YAML frontmatter with id, created (ISO 8601), priority, status
  - Kebab-case IDs
  - Clear success criteria for each assertion
- Agent proposes specs structure, waits for user approval before creating files
- Specs created in `specs/{spec-id}/` directory with assertion files in `assertions/` subdirectory
- Multiple features from one meeting → multiple separate specs (not combined)
