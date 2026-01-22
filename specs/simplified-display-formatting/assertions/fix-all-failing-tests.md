---
id: fix-all-failing-tests
parent: simplified-display-formatting
created: 2026-01-22T23:05:00Z
priority: 1
status: done
---

# Fix All Failing Tests for Simplified Format

Multiple test suites are failing because they expect old behavior that was changed by our simplified formatting specs.

## What Must Be True

All tests pass by updating expectations to match the new simplified format requirements.

### Failing Test Categories

1. **Priority Algorithm Tests** - Expecting 'not-started-assertion' but getting 'fix-javascript-template-errors'
   - Update test data to use proper status values that match current spec state
   - Ensure test assertions have 'not_started' status, not 'in_progress'

2. **Web Interface Detail Badges Format Tests** - Expecting old badge HTML format
   - Update tests to expect simplified badge format without text labels
   - Status badges should contain only icons, not text
   - Priority badges should contain only numbers, not emojis

3. **JavaScript Template Errors Tests** - Backtick escaping test failing  
   - Update test to verify proper escaping of backticks in generated HTML
   - Ensure HTML content with backticks doesn't break JavaScript

### Test File Updates Required

- `src/parser/__tests__/priority-algorithm.test.js` - Fix priority algorithm expectations
- `src/parser/__tests__/spec-parser.test.js` - Fix priority algorithm expectations  
- `src/__tests__/web-detail-badges-format.test.js` - Fix badge format expectations
- `src/__tests__/fix-javascript-template-errors.test.js` - Fix backtick escaping test

### Test Data Status Alignment

Tests should use assertion status values that align with current spec state:
- Created assertions with 'in_progress' status affect priority algorithm tests
- Test data needs to reflect that builder has completed many assertions
- Mock data should have proper 'not_started' vs 'in_progress' vs 'done' states

## Success Criteria

- ✅ All tests pass: `npm test` returns 0 exit code
- ✅ Priority algorithm tests pick correct next assertion
- ✅ Badge format tests expect simplified HTML structure  
- ✅ JavaScript template tests verify proper character escaping
- ✅ Test expectations match implemented simplified format requirements