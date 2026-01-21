---
id: receives-user-requests
parent: coach-agent
created: 2026-01-20T17:05:00Z
priority: 3
status: done
---

# Coach Must Receive and Parse User Requests

**Implementation:** app/coach/index.js, app/coach/cli.js

## What Must Be True

The coach agent must accept natural language input from users describing what they want to build or change.

## Input Formats

The coach should handle various input styles:
- **Direct requests**: "Add dark mode"
- **Problem statements**: "Users can't see the dashboard in bright sunlight"
- **Feature descriptions**: "We need a way for users to export their data"
- **Change requests**: "Fix the copy on the login page - it should say 'Sign in' not 'Login'"
- **Vague ideas**: "Make the app faster"

## What It Does

1. **Accept input** via command line or conversational interface
2. **Parse intent** - understand what the user wants
3. **Identify type**:
   - New feature (create new spec)
   - Update existing feature (modify spec)
   - Bug fix (may update existing assertion)
   - Refactor (may update multiple specs)

## Success Criteria

- ✅ Coach accepts text input
- ✅ Coach handles various input formats
- ✅ Coach identifies if request is new spec or update
- ✅ Coach can handle vague requests (will ask clarifying questions)
- ✅ Coach responds appropriately to user's intent
