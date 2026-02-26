---
id: searchbar-filters-spec-tree
parent: spec-explorer-web-interface
created: 2026-02-26T20:00:00Z
priority: 2
status: not_started
branch: feature/spec-searchbar
---

# Searchbar Filters Spec Tree

## What Must Be True

A text input in the tree panel allows real-time filtering of specs and assertions by name, content, and status.

## Success Criteria

- Search input appears in the tree panel header, below the "Show completed specs" toggle and above the spec tree list
- Placeholder text reads "Search specs..."
- Typing filters the spec tree in real-time (case-insensitive substring match)
- Search matches against spec names, assertion names, status, and priority
- Search overrides the "hide completed" toggle — matching completed specs become visible during an active search
- Clearing the search input restores normal toggle behavior (completed specs re-hide if toggle is unchecked)
- If an assertion matches, its parent spec is shown and expanded to reveal the match
- If only a spec name matches, it shows with all its assertions visible
- Specs and assertions that don't match are hidden
- Empty search input = no filter applied (all specs visible per toggle state)

## Visual Example

**Search input placement:**
```
┌─────────────────────────────────┐
│ Spec Tree - project             │
│ 5 specs, 12 assertions          │
│ (3 completed specs hidden)      │
│ ☐ Show completed specs          │
│                                 │
│ [🔍 Search specs...           ] │
│                                 │
│ ▸ authentication (in_progress)  │
│ ▸ dashboard (not_started)       │
└─────────────────────────────────┘
```

**With active search "parser":**
```
┌─────────────────────────────────┐
│ [🔍 parser                    ] │
│                                 │
│ ▾ spec-parser (done)            │  ← completed but visible because it matches
│   ├ parser-reads-frontmatter    │
│   └ parser-handles-nested       │
│ ▾ robust-error-handling         │  ← visible because child matches
│   └ parser-error-messages       │  ← matches "parser"
└─────────────────────────────────┘
```

## Behavior Rules

- Filtering is purely visual (CSS class toggling + JS), no data re-parsing
- Follows existing patterns: event delegation, classList manipulation, vanilla JS
- Search state does NOT persist in localStorage (fresh on each page load)
- When search is active and matches a completed spec, that spec is shown regardless of toggle state
- When search is cleared, the completed specs toggle reasserts control
