---
id: update-failing-tests
parent: simplified-display-formatting
created: 2026-01-22T22:40:00Z
priority: 1
status: done
---

# Update Failing Tests for Simplified Format

The Detail Badges Format test suite is failing because it expects the old HTML format with text labels and emojis.

## What Must Be True

Tests validate the new simplified badge format instead of the old verbose format.

### Test Updates Required

File: `src/__tests__/detail-badges-format.test.js`

- **generateDetailStatusBadge tests** should expect: `<span class="detail-status-badge status-${status}">${icon}</span>`
- **generateDetailPriorityBadge tests** should expect: `<span class="detail-priority-badge priority-${priority}">${priority}</span>`
- **Status badge tests** should verify icon-only content (no text labels)
- **Priority badge tests** should verify number-only content (no emojis)

### Expected Test Behavior

- Status badges contain only status icons (✅, 🚧, 📋, etc.)
- Priority badges contain only numbers (1, 2, 3)
- HTML structure matches our simplified format specification
- All 4 failing test cases should pass

## Success Criteria

- ✅ Run `npm test` and all Detail Badges Format tests pass
- ✅ Tests validate simplified format: priority number, status icon, title
- ✅ No text labels or emoji decorations in test expectations  
- ✅ CI pipeline passes completely