---
id: simplified-display-formatting
created: 2026-01-22T22:30:00Z
priority: 2
---

# Simplified Display Formatting

Status and priority displays should be simplified to reduce visual noise and improve readability.

## Requirements

The current status and priority displays are too verbose with unnecessary text labels and emoji clutter. This spec defines a cleaner, more focused display format.

### Display Format

All status and priority displays should follow this pattern:
**Priority Number → Status Icon → Title**

### Status Display Rules

- Status should show **icon only** (no text labels)
- Remove text like "Done", "In Progress", "Not Started" 
- Keep existing status icons: ✅ 🚧 📋 ⏸️ 📝

### Priority Display Rules  

- Priority should show **number only** (no emoji decorations)
- Remove priority emojis like 🔥 ⚠️ 💡
- Show just: 1, 2, 3

### Affected Components

This applies to:
- Console status command output
- Web interface tree view
- Web interface detail badges
- All other status/priority displays in the system

## Success Criteria

- All displays show format: `{priority} {status_icon} {title}`
- No text labels on status badges
- No emoji decorations on priority badges  
- Consistent ordering across all components