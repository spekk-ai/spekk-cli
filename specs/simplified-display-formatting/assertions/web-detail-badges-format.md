---
id: web-detail-badges-format  
parent: simplified-display-formatting
created: 2026-01-22T22:33:00Z
priority: 2
status: done
---

# Web Interface Detail Badges Format

The detail panel badges in the spec explorer show simplified status and priority.

## What Must Be True

Detail view meta badges display simplified format without text labels or emoji decorations.

### Status Detail Badge
- Function `generateDetailStatusBadge()` returns: `<span class="detail-status-badge">{status_icon}</span>`
- No text labels like "done", "in progress", "not started"
- Only the status icon (✅, 🚧, 📋, etc.)

### Priority Detail Badge  
- Function `generateDetailPriorityBadge()` returns: `<span class="detail-priority-badge">{priority}</span>`
- No emoji decorations like 🔥, ⚠️, 💡
- Only the priority number (1, 2, 3)

### Meta Display Format
- Detail meta shows: `Status: {icon}` and `Priority: {number}`
- Consistent with simplified format across the system

**Tests:** src/__tests__/detail-badges-format.test.js

## Success Criteria

- ✅ Run `spekk show` and click on any spec/assertion
- ✅ Status badge shows only icon (no text)  
- ✅ Priority badge shows only number (no emoji)
- ✅ Functions return simplified HTML markup
- ✅ Detail view matches tree view formatting style