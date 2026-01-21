---
id: enforces-folder-structure
parent: spec-parser
created: 2026-01-20T16:00:00Z
priority: 1
status: done
---

# Parser Must Enforce Folder Structure

## What Must Be True

All specs in `specs/` must follow the declared folder structure:

```
specs/
├── {spec-id}/
│   ├── {spec-id}.md              # Main spec file
│   └── assertions/               # Assertions subfolder
│       ├── {assertion-id}.md
│       └── ...
```

## Validation Rules

The parser must validate that:

1. **Each spec is a folder** - No flat `.md` files at `specs/*.md` level
2. **Main spec file exists** - `specs/{spec-id}/{spec-id}.md` must exist
3. **Assertions folder exists** - `specs/{spec-id}/assertions/` must exist (can be empty)
4. **Assertion files in correct location** - All assertion `.md` files must be inside `assertions/` subfolder
5. **Consistent naming** - Folder name should match the spec file name (both use same `id`)

## Current State

Existing specs are currently flat files:
- `specs/spec-format.md` ❌
- `specs/living-spec-dashboard.md` ❌
- `specs/spec-builder-prototype.md` ❌

These must be migrated to:
- `specs/living-spec-dashboard/`
  - `living-spec-dashboard.md`
  - `assertions/`
- `specs/spec-builder-prototype/`
  - `spec-builder-prototype.md`
  - `assertions/`

## Migration Requirements

When this assertion is worked on:

1. Create folder for each existing flat spec
2. Move spec file into its folder
3. Create `assertions/` subfolder
4. Extract any inline assertions from spec files into separate assertion files
5. Update any cross-references or links

## Special Case: spec-format.md

The content of `spec-format.md` should become assertions under `spec-parser/assertions/` because the format rules ARE validation criteria for the parser:
- `parses-frontmatter.md`
- `validates-required-fields.md`
- `validates-timestamps.md`
- `validates-status-values.md`
- `detects-duplicate-ids.md`

## Success Criteria

This assertion is "done" when:
- ✅ All specs in `specs/` follow folder structure
- ✅ No flat `.md` files at `specs/*.md` level (except index if we add one later)
- ✅ Parser validates structure and rejects malformed specs
- ✅ All existing spec content preserved and functional
