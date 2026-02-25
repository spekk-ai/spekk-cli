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

Stations (assertion dots) in the branch metro map are clickable and navigate to that assertion's detail view.

## Success Criteria

- ✅ Clicking a station in metro map shows that assertion's detail
- ✅ Metro map updates to highlight the newly selected assertion
- ✅ Tree view (left panel) updates to show the clicked assertion as selected
- ✅ If clicked assertion's parent spec is collapsed, expand it
- ✅ Smooth transition between assertion views (no page reload)
- ✅ Cursor changes to pointer on station hover
- ✅ Hover shows tooltip with assertion title (optional enhancement)

## Behavior Example

User viewing `chat-session-model`:
1. Sees metro map showing all 4 assertions in feature/chat-system
2. Clicks `chat-message-input` station in metro map
3. Detail panel updates to show `chat-message-input` content
4. Metro map updates: `chat-message-input` now highlighted
5. Tree view highlights `chat-message-input` item

## Implementation Notes

- Reuse existing `showDetail()` JavaScript function
- Add click handlers to station `<circle>` elements
- Pass assertion ID to `showDetail()`
- Update metro map by toggling CSS classes for highlight
- Station elements need `data-assertion-id` attributes
- Prevent event bubbling to avoid conflicts with parent clicks
