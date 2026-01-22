---
id: assertions-appear-as-subitems
parent: spec-explorer-web-interface
created: 2026-01-22T21:00:00Z
priority: 2
status: not_started
---

# Assertions Appear as Sub-items

## Assertion

Assertions are displayed as sub-items under their parent specs in the tree structure.

## Success Criteria

- When a spec is expanded, its assertions appear as child nodes
- Assertions are visually distinguished from specs (indentation, styling, icons)
- Assertions are organized under the correct parent spec
- Empty specs (with no assertions) can still be expanded but show appropriate messaging

## Test Plan

- Expand a spec in the tree interface
- Verify assertions appear as indented sub-items
- Check that assertion names match those in the specs/[spec]/assertions/ directory
- Confirm visual hierarchy is clear and intuitive