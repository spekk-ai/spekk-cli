---
id: creates-spec-files
parent: coach-agent
created: 2026-01-20T17:15:00Z
priority: 3
status: done
---

# Coach Must Create Well-Formed Spec Files

**Tests:** app/coach/__tests__/creates-spec-files.test.js

## What Must Be True

After refinement dialogue, coach must create properly formatted spec files that follow the declared structure.

## File Structure It Creates

```
specs/
└── {spec-id}/
    ├── {spec-id}.md              # Main spec file
    └── assertions/
        ├── {assertion-1}.md
        ├── {assertion-2}.md
        └── ...
```

## Spec File Format

Main spec file must have:
```yaml
---
id: spec-id
created: 2026-01-20T17:15:00Z
priority: 2
status: not_started
---

# Spec Title

## Overview
What this spec defines...

## What It Does
Detailed description...
```

## Assertion File Format

Each assertion must have:
```yaml
---
id: assertion-id
parent: spec-id
created: 2026-01-20T17:16:00Z
priority: 1
status: not_started
---

# Assertion Title

## What Must Be True
Clear, testable requirement...

## Success Criteria
- ✅ Criterion 1
- ✅ Criterion 2
```

## Validation

Coach must ensure:
- **Valid YAML frontmatter** (parseable, correct fields)
- **Unique IDs** (no duplicates)
- **Valid priorities** (1, 2, or 3)
- **ISO 8601 timestamps** (YYYY-MM-DDTHH:MM:SSZ)
- **Kebab-case IDs** (lowercase-with-hyphens)
- **Clear success criteria** (how to validate assertion)
- **Testability noted** (can this be automated?)

## Success Criteria

- ✅ Coach creates folder structure correctly
- ✅ Coach writes valid YAML frontmatter
- ✅ Coach generates unique, valid IDs
- ✅ Coach includes clear success criteria
- ✅ Coach sets appropriate priorities
- ✅ Coach creates testable assertions
- ✅ Generated specs pass parser validation
