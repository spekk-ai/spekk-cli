---
id: branch-metro-map-in-detail-panel
parent: spec-explorer-web-interface
created: 2026-02-25T19:00:00Z
priority: 1
status: done
branch: feature/dependency-visualization
---

# Persistent Collapsible Metro Map in Detail Panel

## What Must Be True

The detail panel is split horizontally: a collapsible metro map section at the top, and scrollable assertion content below. The metro map persists across station clicks (no re-render), gives the dependency graph full panel width, and can be collapsed when the user wants to focus on content.

## Success Criteria

- 2-column layout: Tree (left) | Detail (right) — same as original
- Detail panel split horizontally: metro map top, assertion content bottom
- Metro map section takes full width of detail panel (~800px)
- Metro map renders once per branch, NOT re-rendered when clicking stations
- Clicking a station updates assertion content below AND toggles highlight class on map — no SVG re-render
- Pan/zoom state preserved when navigating between assertions in same branch
- Collapsible: toggle button collapses metro map to a thin "Branch Dependencies ▶" bar
- Collapsed state saved in localStorage
- When assertion has no applicable metro map (main branch, no deps), metro section shows the no-dependencies notice or is hidden
- When switching to an assertion on a different branch, metro map re-renders for new branch
- Compact vertical spacing: ~45px between independent trees (down from 70px)
- Labels show assertion titles below each station

## Visual Structure

**Expanded (default):**
```
╔═══════════════╦═════════════════════════════════════════╗
║ Tree Panel    ║ ▼ Branch Dependencies              [−] ║
║               ║ ○───○───◉───○───●                      ║
║ ▼ spec-a      ║      └──○───○───●                      ║
║   assertion-1 ╠═════════════════════════════════════════╣
║   assertion-2 ║ Chat Message Input                     ║
║ ▼ spec-b      ║ Status: not_started  Priority: 2       ║
║   assertion-3 ║                                        ║
║               ║ ## What Must Be True                   ║
║               ║ [content...]                           ║
╚═══════════════╩═════════════════════════════════════════╝
```

**Collapsed:**
```
╔═══════════════╦═════════════════════════════════════════╗
║ Tree Panel    ║ ▶ Branch Dependencies              [+] ║
║               ╠═════════════════════════════════════════╣
║ ▼ spec-a      ║ Chat Message Input                     ║
║   assertion-1 ║ Status: not_started  Priority: 2       ║
║   assertion-2 ║                                        ║
║ ▼ spec-b      ║ ## What Must Be True                   ║
║   assertion-3 ║ [content...]                           ║
╚═══════════════╩═════════════════════════════════════════╝
```

## Layout Rules

**Container:**
- Keep existing `max-width: 1200px` container
- Tree panel: ~400px fixed width (unchanged)
- Detail panel: flex: 1

**Detail panel structure:**
- `.metro-map-section` at top: collapsible, with pan-and-zoom
- `.detail-content-section` below: scrollable, fills remaining height
- Divider border between sections

**Metro map section (expanded):**
- Max-height: ~300px (constrains vertical space)
- `overflow: hidden` with pan-and-zoom for navigation
- Background: `#f8fafc`
- Collapsible header bar with toggle arrow and "Branch Dependencies" label
- Full width of detail panel for generous horizontal space

**Compact spacing:**
- Tree spacing (vertical between independent chains): 45px
- Layer spacing (horizontal between dependency levels): 120px
- Station radius: 8px (normal), 10px (selected)

## Collapse Behavior

- Toggle button in header bar: `▼`/`▶` arrow or `[−]`/`[+]` icon
- Collapsed: metro map section shrinks to header bar only (~36px height)
- Expanded: metro map section shows at configured max-height
- Transition: smooth height animation
- State persisted: `localStorage.setItem('spekkMetroMapCollapsed', 'true/false')`
- Default: expanded

## Implementation Notes

- Generate ONE metro map SVG per branch, embedded in `.metro-map-section`
- On `showDetail()`, update `.detail-content-section` content AND toggle `.metro-station-current` class on stations
- The metro map SVG element stays in the DOM across `showDetail()` calls
- When assertion's branch changes, regenerate the metro map SVG
- Pan-and-zoom handlers attached to `.metro-map-section` container
- Metro map section header shows branch name: "Branch Dependencies — feature/dependency-visualization"
