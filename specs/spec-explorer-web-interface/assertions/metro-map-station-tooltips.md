---
id: metro-map-station-tooltips
parent: spec-explorer-web-interface
created: 2026-02-25T20:03:00Z
priority: 2
status: done
depends-on: branch-metro-map-in-detail-panel
branch: feature/dependency-visualization
---

# Metro Map Station Tooltips

## What Must Be True

Hovering over a metro station shows a tooltip with the full assertion title, since titles are truncated in the visual.

## Success Criteria

- ✅ Tooltip appears on station hover
- ✅ Shows full assertion title (not truncated)
- ✅ Positioned above the station dot
- ✅ Styled with dark background, white text, rounded corners
- ✅ Smooth fade in/out animation
- ✅ Tooltip doesn't cause layout shift or shake
- ✅ Disappears when mouse leaves station

## Visual Example

```
     ┌─────────────────────────┐
     │ chat-message-input      │  ← Tooltip
     └─────────┬───────────────┘
               ↓
              ◉  ← Station (hovered)
          chat-me...
```

## Implementation

**CSS:**
```css
.metro-station {
  position: relative;
}

.metro-station-tooltip {
  position: absolute;
  bottom: calc(100% + 10px);
  left: 50%;
  transform: translateX(-50%);
  background: #1e293b;
  color: white;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 11px;
  white-space: nowrap;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.2s;
  z-index: 100;
}

.metro-station:hover .metro-station-tooltip {
  opacity: 1;
}

.metro-station-tooltip::after {
  /* Arrow pointing down */
  content: '';
  position: absolute;
  top: 100%;
  left: 50%;
  transform: translateX(-50%);
  border: 5px solid transparent;
  border-top-color: #1e293b;
}
```

**HTML in SVG:**
Option 1 - Use `<title>` element (native SVG tooltip):
```xml
<g class="metro-station">
  <title>Full assertion title here</title>
  <circle class="station-dot" r="8"/>
  <text>Truncated...</text>
</g>
```

Option 2 - Use HTML overlay (more control):
- Add `data-title="Full title"` to `<g>` element
- JavaScript creates HTML tooltip on hover
- Position based on SVG element coordinates

## Implementation Notes

- SVG `<title>` is easiest but has limited styling
- HTML overlay gives more control over appearance
- Ensure tooltip doesn't interfere with hover shake fix
- Use `pointer-events: none` on tooltip to prevent hover conflicts
- Consider max-width and text wrapping for very long titles
