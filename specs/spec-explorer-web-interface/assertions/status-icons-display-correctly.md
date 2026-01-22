---
id: status-icons-display-correctly
parent: spec-explorer-web-interface
created: 2026-01-22T22:20:00Z
priority: 2
status: not_started
---

# Status Icons Display Correctly

## Description

The spec explorer displays distinct, meaningful icons for each status type to provide clear visual differentiation in the tree interface.

## Requirements

**Status Icon Mapping:**
- `not_started`: Gray empty circle (○) - represents unbegun work
- `in_progress`: Spinning arrow (🔄) - represents active work 
- `done`: Green checkmark (✅) - represents completed work
- `failed`: Red X (❌) - represents blocked work
- `draft`: Pause icon (⏸️) - represents planning/placeholder items

**Icon Display:**
- Icons appear consistently in both tree badges and detail panel
- Each status has appropriate CSS styling to match the icon meaning
- Draft items are visually distinct from not_started items

## Success Criteria

- [ ] All five status values have defined icons in `getStatusIcon()` function
- [ ] CSS classes exist for all status types including `.status-draft`
- [ ] Draft specs/assertions display pause icon (⏸️) in tree view
- [ ] Not_started specs/assertions display gray circle (○) in tree view
- [ ] Icons are consistent between tree view and detail panel views

## Testing

Create test specs with each status type and verify icons display correctly in the web interface.