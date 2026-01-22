---
id: fix-javascript-template-errors
parent: simplified-display-formatting
created: 2026-01-22T22:58:00Z
priority: 1
status: done
---

# Fix JavaScript Template Generation Errors

The spekk show command generates broken HTML with JavaScript syntax errors and missing functions.

## What Must Be True

Generated HTML contains valid JavaScript without syntax errors or missing function definitions.

### Specific JavaScript Errors to Fix

1. **"Unexpected token '{'""** - Template literal syntax issue in HTML generation
2. **"toggleSpec is not defined"** - Missing JavaScript functions in generated HTML
3. **Missing showDetail function** - Required for click handlers

### Root Causes

- Special characters in spec content breaking JavaScript string interpolation  
- Template literals not properly escaped in HTML generation
- JavaScript functions missing from the generated HTML template

### Required Fixes

- **Escape spec data** properly when injecting into JavaScript template strings
- **Include all JavaScript functions** (toggleSpec, showDetail) in generated HTML
- **Validate generated HTML** has no syntax errors before writing to file
- **Handle special characters** in spec titles, IDs, and content that could break JS

### Test Cases

Generated HTML should work correctly when specs contain:
- Single quotes in titles: `User's Dashboard`  
- Double quotes in content: `"quoted text"`
- Backticks in code: `` `code example` ``
- Curly braces: `{example: value}`

**Tests:** src/__tests__/fix-javascript-template-errors.test.js

## Success Criteria

- ✅ Run `spekk show` in `/Users/william/thinknimble/spekk/` 
- ✅ Generated HTML has no JavaScript console errors
- ✅ `toggleSpec` and `showDetail` functions are defined and working
- ✅ Click handlers on spec items work correctly
- ✅ Special characters in spec data don't break JavaScript