---
id: yaml-frontmatter-updates
parent: coordinator-skill
created: 2026-02-25T00:00:00Z
priority: 1
status: not_started
depends-on: branch-assignment
---

# Coordinator Updates YAML Frontmatter with Dependencies and Branches

Coordinator writes `depends-on` and `branch` fields to assertion YAML frontmatter and commits changes.

## Success Criteria

### YAML Updates

For each assertion in the coordination plan:

**Add `depends-on` field (only if dependency exists):**
```yaml
---
id: chat-message-input
parent: chat-system
created: 2026-02-24T00:00:00Z
priority: 2
status: not_started
depends-on: chat-session-model
---
```

**Add `branch` field:**
```yaml
---
id: chat-message-input
parent: chat-system
created: 2026-02-24T00:00:00Z
priority: 2
status: not_started
depends-on: chat-session-model
branch: feature/chat-system
---
```

**Assertions with no dependencies:**
```yaml
---
id: websocket-connection
parent: chat-system
created: 2026-02-24T00:00:00Z
priority: 1
status: not_started
branch: feature/chat-system
---
# No depends-on field (omitted = no dependency)
```

### Field Placement

Insert new fields in logical order:
```yaml
id: ...
parent: ...          # (if assertion)
created: ...
priority: ...
status: ...
depends-on: ...      # NEW (if exists)
branch: ...          # NEW
```

### Sparse Updates

- **Only add `depends-on` if there IS a dependency**
- Omitted field = no dependency (same as `depends-on: null`)
- **All assertions get `branch` field**

### Commit Strategy

Single commit with all YAML updates:

```
Add coordinator dependency and branch metadata

Applied coordinator skill to organize work:

feature/chat-system (4 assertions):
  - websocket-connection → chat-session-model → chat-message-input
  - user-presence-tracking

feature/clinical-trials (2 assertions):
  - trial-search-api → trial-eligibility-check

main (2 assertions):
  - update-button-styles
  - fix-header-typo

Changes:
- Added depends-on field where dependencies exist
- Added branch field to all assertions
- No changes to spec content or existing metadata
```

### Preserve Existing Content

- **Do not modify** existing YAML fields
- **Do not modify** markdown content below frontmatter
- **Preserve formatting** (indentation, blank lines)
- **Maintain order** of existing fields

## Implementation Notes

### YAML Writing Strategy

Use existing parser utilities:
```javascript
import { parseFrontmatter } from './parser/index.js';

function updateAssertionFrontmatter(filePath, updates) {
  const content = fs.readFileSync(filePath, 'utf8');
  const { data, content: markdown } = parseFrontmatter(content);
  
  // Merge updates (only add new fields)
  const updated = {
    ...data,
    ...updates  // { depends-on: 'parent-id', branch: 'feature/name' }
  };
  
  // Reconstruct YAML
  const yaml = stringifyYAML(updated);
  const newContent = `---\n${yaml}---\n${markdown}`;
  
  fs.writeFileSync(filePath, newContent, 'utf8');
}
```

### YAML Serialization

Maintain field order:
1. `id`
2. `parent` (if present)
3. `created`
4. `priority`
5. `status`
6. `depends-on` (if present)
7. `branch`
8. Other fields (preserve existing order)

### Validation Before Writing

- Verify all dependency IDs exist
- Verify no circular dependencies
- Verify branch names are valid (kebab-case)
- Dry-run: show diff before writing

## User Experience

### Confirmation Step

Before writing files:
```
Ready to update 8 assertion files with dependency and branch metadata.

Files to update:
  specs/chat-system/assertions/websocket-connection.md
  specs/chat-system/assertions/chat-session-model.md
  specs/chat-system/assertions/chat-message-input.md
  ... (5 more)

Proceed? [y/N]
```

### Summary After Commit

```
✅ Updated 8 assertion files
✅ Committed changes: abc123f

Next steps:
1. Create feature branches:
   git checkout -b feature/chat-system
   git checkout -b feature/clinical-trials

2. Start building:
   git checkout feature/chat-system
   spekk builder --once
```

## Validation

- YAML frontmatter updated correctly in all affected files
- New fields inserted in correct position
- Existing fields and content preserved
- Single commit created with clear message
- Dry-run shows accurate preview
- User can confirm or cancel before writing

**Tests:** 
- Unit tests for YAML parsing/writing
- Integration tests with sample assertion files
- Verify no content corruption
