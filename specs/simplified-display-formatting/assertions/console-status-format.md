---
id: console-status-format
parent: simplified-display-formatting
created: 2026-01-22T22:31:00Z
priority: 2
status: not_started
---

# Console Status Display Format

The `spekk status` command displays specs and assertions with simplified formatting.

## What Must Be True

Console output shows status and priority in the format: `{priority} {status_icon} {title}`

### Spec Lines Format
- Spec display: `{priority} {status_icon} {spec_title} (x/y assertions complete)`
- No text labels like "Done", "In Progress" 
- No priority emojis like 🔥, ⚠️, 💡

### Assertion Lines Format  
- Assertion display: `  {priority} {status_icon} {assertion_title}`
- Indented with 2 spaces
- No text labels on status
- No emoji decorations on priority

### Next Priority Item Format
- Status line shows: `Status: {status_icon}` (no text label)
- Priority shows number only

## Success Criteria

- ✅ Run `spekk status` and verify format matches specification
- ✅ Spec lines show: `1 ✅ Example Spec (2/3 assertions complete)`
- ✅ Assertion lines show: `  2 🚧 Example Assertion`
- ✅ Next item shows: `Status: ✅` (not `Status: ✅ done`)