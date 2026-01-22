---
id: fix-javascript-template-errors
parent: simplified-display-formatting
created: 2026-01-22T22:58:00Z
priority: 2
status: done
---

# Fix JavaScript Template Generation Errors

The spekk show command should eliminate JavaScript escaping issues by using modern event delegation instead of inline event handlers.

## What Must Be True

Generated HTML contains valid JavaScript without syntax errors by avoiding inline event handlers and using data attributes instead.

### Previous Problems (Now Eliminated)

1. ~~**"Unexpected token '{'""** - Template literal syntax issue~~ → **Solved by removing inline handlers**
2. ~~**Quote escaping in onclick attributes**~~ → **Solved by using data attributes** 
3. **Missing JavaScript functions** - `toggleSpec` and `showDetail` functions still needed

### New Approach: Event Delegation

- **Replace all inline onclick handlers** with data attributes
- **Use single document event listener** for all interactions
- **No more string escaping issues** - data attributes are safe for any content
- **Include JavaScript functions** (toggleSpec, showDetail) but called via event delegation

### Required Implementation

**Replace this pattern:**
```html
onclick="toggleSpec('spec-parser', event)"
onclick="showDetail('assertion-id', 'assertion', event)" 
```

**With this pattern:**
```html
data-action="toggle-spec" data-spec-id="spec-parser"
data-action="show-detail" data-assertion-id="assertion-id"
```

### Test Cases

Generated HTML should work correctly when specs contain any characters:
- Single quotes in titles: `User's Dashboard`  
- Double quotes in content: `"quoted text"`
- Backticks in code: `` `code example` ``
- Curly braces: `{example: value}`
- **No escaping needed** - data attributes handle all content safely

**Tests:** src/__tests__/fix-javascript-template-errors.test.js

## Success Criteria

- ✅ Run `spekk show` and generated HTML has no JavaScript console errors
- ✅ `toggleSpec` and `showDetail` functions are defined and working 
- ✅ All HTML uses data attributes instead of inline onclick handlers
- ✅ Event delegation handles all user interactions via document listener
- ✅ Click handlers work correctly regardless of special characters in spec data
- ✅ No inline event handlers exist in generated HTML
- ✅ Data attributes safely handle any spec content without escaping issues