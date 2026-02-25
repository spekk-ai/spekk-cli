---
id: displays-dependency-info-inline
parent: spec-explorer-web-interface
created: 2026-02-25T18:00:00Z
priority: 1
status: draft
---

# Displays Dependency Info Inline

## What Must Be True

When an assertion has a `depends-on` field in its YAML frontmatter, the spec explorer tree view displays this dependency information directly under the assertion.

## Success Criteria

- ✅ Assertions with `depends-on` field show dependency info below the assertion title
- ✅ Format: "→ depends on: {assertion-id}" in smaller, gray text
- ✅ Assertions without dependencies show no additional text
- ✅ Multiple assertions can depend on the same parent (shows independently)
- ✅ Dependency text is indented slightly from the assertion
- ✅ Visual style is subtle (doesn't compete with assertion titles)

## Example

```
spec-parser/
  ├─ parses-frontmatter (priority 1) ✅
  ├─ validates-fields (priority 1) ✅
  │  → depends on: parses-frontmatter
  ├─ outputs-json (priority 2) 🔄
  │  → depends on: validates-fields
```

## Implementation Notes

- Parser already exposes `depends-on` field from assertion YAML
- Add dependency info to HTML generation in `src/show/cli.js`
- Style with `color: #64748b; font-size: 12px; margin-left: 20px;`
