---
id: fixed-width-markdown-font
parent: simplified-display-formatting
created: 2026-01-22T22:50:00Z
priority: 2
status: not_started
---

# Fixed-Width Font for Markdown Content

Spec and assertion markdown content should use a fixed-width font for better readability and code-like formatting.

## What Must Be True

Detail panel markdown content uses monospace font family for consistent character spacing.

### Font Family Requirements

- **Primary font:** Consolas (Windows/Office)
- **Fallback fonts:** Monaco, 'Courier New', monospace
- Apply to all markdown content in detail view
- Maintain readability while providing fixed-width spacing

### CSS Target

`.detail-body` and `.detail-body pre` elements should use:
```css
font-family: Consolas, Monaco, 'Courier New', monospace;
```

### Content Areas Affected

- Spec descriptions and content
- Assertion descriptions and success criteria  
- All markdown text in the detail panel
- Preserve existing `<pre>` tag styling but with better font

## Success Criteria

- ✅ Run `spekk show` and click on any spec/assertion
- ✅ Markdown content displays in fixed-width font (Consolas/Monaco)
- ✅ Text remains readable with proper line spacing
- ✅ Code blocks and regular text both use monospace font