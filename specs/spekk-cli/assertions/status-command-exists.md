---
id: status-command-exists
parent: spekk-cli
created: 2026-01-21T19:02:00Z
priority: 2
status: done
---

# Status Command Exists

**Tests:** src/__tests__/status-command.test.js

## Requirement

Status command shows comprehensive overview of all specs and assertions with their completion status. Available both as `spekk status` and `npm run status`.

## Success Criteria

### Command Functionality:
✅ `spekk status` runs without errors
✅ `npm run status` runs without errors
✅ Shows all specs found in current directory's specs/ folder
✅ Displays assertion count and completion ratio for each spec
✅ Lists individual assertions with status icons
✅ Shows next priority item at bottom

### Output Format:
✅ Uses clear status icons (✅ done, 🚧 in progress, 📋 not started, ⏸️ blocked)
✅ Groups assertions under parent specs with indentation
✅ Shows completion ratios (e.g., "2/5 assertions complete")
✅ Highlights next priority item clearly
✅ Handles empty specs/ directory gracefully

### Performance:
✅ Executes in < 100ms even with many specs
✅ Parses all specs and assertions correctly
✅ Uses same validation logic as main parser