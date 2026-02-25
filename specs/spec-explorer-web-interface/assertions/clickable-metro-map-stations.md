---
id: clickable-metro-map-stations
parent: spec-explorer-web-interface
created: 2026-02-25T19:01:00Z
priority: 2
status: done
depends-on: branch-metro-map-in-detail-panel
branch: feature/dependency-visualization
---

# Clickable Metro Map Stations

## What Must Be True

Stations in the persistent metro map column are clickable and navigate to that assertion's detail view without re-rendering the metro map SVG or losing pan/zoom state.

## Success Criteria

- Clicking a station updates the detail panel (center column) to show that assertion
- Metro map SVG is NOT re-rendered — only CSS classes are toggled
- Previously selected station loses highlight; newly selected station gains highlight
- Selected station: glow effect + thicker stroke (via `.metro-station-current` class)
- Pan/zoom position is preserved across station clicks
- Tree view (left panel) updates to show the clicked assertion as selected
- If clicked assertion's parent spec is collapsed in tree, expand it
- Cursor changes to pointer on station hover

## Behavior Example

User viewing `chat-session-model`:
1. Sees persistent metro map in right column with all branch assertions
2. Clicks `chat-message-input` station in metro map
3. Detail panel (center) updates to show `chat-message-input` content
4. Metro map stays in place — only the highlight class moves to new station
5. Pan/zoom position unchanged
6. Tree view highlights `chat-message-input` item

## Implementation Notes

- `showDetail()` already updates detail panel — add class toggling logic:
  ```javascript
  // Remove current highlight from all stations
  document.querySelectorAll('.metro-station-current').forEach(el => {
    el.classList.remove('metro-station-current');
  });
  // Add highlight to clicked station
  const station = document.querySelector(`.metro-station[data-assertion-id="${id}"]`);
  if (station) station.classList.add('metro-station-current');
  ```
- Station click handlers use existing event delegation on `data-action="show-detail"`
- No SVG regeneration — the metro map column content stays untouched
- CSS for current station:
  ```css
  .metro-station-current circle {
    stroke: #2563eb;
    stroke-width: 4;
    filter: drop-shadow(0 0 6px rgba(59, 130, 246, 0.6));
  }
  ```
