---
id: nested-spec-organization
created: 2026-01-22T17:15:00Z
priority: 3
status: draft
---

# Nested Spec Organization

Support for grouping related specs into logical sub-projects or feature areas using nested folder structures with group-level specifications.

## Purpose

Allow organizing specs into hierarchical groups for:
- Sub-projects (e.g., mockup creation, API redesign)
- Feature areas (e.g., authentication, dashboard, reporting)
- Development phases (e.g., MVP, v2-features, technical-debt)
- Team ownership (e.g., frontend-specs, backend-specs)

## Proposed Structure

```
specs/
├── mockup-project/
│   ├── mockup-project.md        # Group spec with metadata
│   ├── wireframe-spec/
│   │   ├── wireframe-spec.md
│   │   └── assertions/
│   ├── user-flow-spec/
│   │   ├── user-flow-spec.md  
│   │   └── assertions/
│   └── visual-design-spec/
│       ├── visual-design-spec.md
│       └── assertions/
├── authentication/
│   ├── authentication.md           # Group spec
│   ├── login-flow/
│   ├── password-reset/
│   └── session-management/
└── individual-spec/                 # Non-grouped specs still supported
    ├── individual-spec.md
    └── assertions/
```

## Group Specifications

Group-level markdown files would include:

**YAML Frontmatter:**
- `id`: Group identifier
- `type: group`: Distinguishes from regular specs
- `priority`: Overall group priority
- `status`: Computed from child specs or explicit

**Content:**
- Purpose and scope of the grouped work
- Dependencies between child specs
- Coordination notes and shared context
- Links to external resources (designs, requirements)

## Parser Changes Required

- Detect and parse nested folder structures
- Validate group specifications alongside regular specs
- Support group status rollups (group status computed from children)
- Handle cross-group spec references
- Maintain backward compatibility with flat structure

## Benefits

- Better organization for complex projects
- Logical grouping of related specifications
- Easier navigation and context switching
- Team-based spec ownership
- Progressive disclosure (view groups first, then drill down)

## Implementation Considerations

This is a significant architectural change that affects:
- Spec parser logic and validation rules
- CLI commands and navigation
- Web interface display and filtering
- Status calculation algorithms
- Cross-reference resolution