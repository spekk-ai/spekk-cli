---
id: displays-spec-tree-with-dropdowns
parent: spec-explorer-web-interface
created: 2026-01-22T21:00:00Z
priority: 2
status: done
---

# Displays Spec Tree with Expandable Dropdowns

## Assertion

The generated HTML displays all specs as an expandable tree structure with dropdown functionality using modern event delegation instead of inline event handlers.

## Success Criteria

- Each spec appears as a top-level tree item
- Tree items can be expanded/collapsed to show/hide children
- Visual indicators show expanded/collapsed state (arrows, icons, etc.)
- Tree structure reflects the actual spec hierarchy from the specs/ directory
- **Uses data attributes instead of onclick handlers (e.g., `data-spec-id="spec-parser"`, `data-action="toggle-spec"`)**
- **JavaScript uses event delegation with a single document event listener**
- **No inline onclick attributes in the generated HTML**
- **No "Uncaught SyntaxError" errors in browser console**
- **All dropdown toggles work correctly via event delegation**

## Implementation Approach

### HTML Structure (No Inline Handlers)
```html
<div class="spec-header" data-spec-id="spec-parser" data-action="toggle-spec">
  <span class="toggle-icon" id="toggle-spec-parser">▶</span>
  <!-- ... -->
</div>

<li class="assertion-item" data-assertion-id="enforces-folder-structure" data-action="show-detail">
  <!-- ... -->
</li>
```

### JavaScript Event Delegation
```javascript
document.addEventListener('click', function(event) {
  const action = event.target.closest('[data-action]')?.getAttribute('data-action');
  
  if (action === 'toggle-spec') {
    const specId = event.target.closest('[data-spec-id]').getAttribute('data-spec-id');
    toggleSpec(specId, event);
  } else if (action === 'show-detail') {
    const assertionId = event.target.closest('[data-assertion-id]').getAttribute('data-assertion-id');
    showDetail(assertionId, 'assertion', event);
  }
});
```

## Test Plan

- Open generated HTML in browser
- Verify all specs from specs/ directory appear in tree
- Click to expand/collapse tree items using event delegation
- Confirm visual feedback for interaction states
- **Check browser console for JavaScript errors (should be none)**
- **Verify no onclick attributes exist in generated HTML**
- **Test all dropdown toggles function properly via data attributes**