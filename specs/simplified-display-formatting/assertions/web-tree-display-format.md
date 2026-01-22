---
id: web-tree-display-format
parent: simplified-display-formatting
created: 2026-01-22T22:32:00Z
priority: 2
status: done
---

# Web Interface Tree Display Format

The spec explorer web interface tree view displays items with simplified formatting.

## What Must Be True

Tree view displays specs and assertions in the format: `{priority} {status_icon} {title}`

### Spec Header Format
- HTML renders: `<span class="priority-badge">{priority}</span> <span class="status-badge"></span> {title}`
- Priority badge contains number only (no emojis)
- Status badge shows icon via CSS (no text content)
- Order: priority first, then status, then title

### Assertion Item Format  
- HTML renders same pattern for assertion items
- Consistent ordering across specs and assertions
- No text labels in status badges
- No emoji content in priority badges

## Success Criteria

- ✅ Run `spekk show` and inspect tree view HTML
- ✅ Priority badges contain only numbers: `<span>1</span>`, `<span>2</span>`, `<span>3</span>`
- ✅ Status badges have no text content (icons via CSS)
- ✅ Display order is: priority number, status icon, title
- ✅ Format is consistent between specs and assertions