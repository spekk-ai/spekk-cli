---
id: clickable-dependency-links
parent: spec-explorer-web-interface
created: 2026-02-25T18:03:00Z
priority: 2
status: draft
---

# Clickable Dependency Links

## What Must Be True

When dependency information is displayed inline ("→ depends on: X"), the dependency name is a clickable link that navigates to the referenced assertion in the tree and shows its detail.

## Success Criteria

- ✅ Dependency text is styled as a link (blue, underlined on hover)
- ✅ Clicking dependency link:
  - Expands parent spec of target assertion (if collapsed)
  - Scrolls to target assertion in tree view
  - Highlights target assertion temporarily (fade animation)
  - Shows target assertion detail in right panel
- ✅ Invalid dependency IDs (broken references) show as plain text with warning color
- ✅ Link cursor changes to pointer on hover
- ✅ Smooth scroll animation to target

## Example Behavior

User clicks "parses-frontmatter" in:
```
validates-fields (priority 1) ✅
→ depends on: parses-frontmatter
```

Result:
1. Spec tree scrolls to `parses-frontmatter` assertion
2. `parses-frontmatter` highlights briefly (yellow background fade)
3. Detail panel shows `parses-frontmatter` content

## Implementation Notes

- Wrap dependency ID in `<a>` tag with click handler
- Use `element.scrollIntoView({ behavior: 'smooth', block: 'center' })`
- Add CSS animation for highlight effect (keyframes fade out)
- Reuse existing `showDetail()` function for displaying assertion
- Validate dependency ID exists before making clickable
