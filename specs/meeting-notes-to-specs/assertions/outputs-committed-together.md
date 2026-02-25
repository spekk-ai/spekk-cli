---
id: outputs-committed-together
parent: meeting-notes-to-specs
created: 2025-02-12T19:30:00Z
priority: 2
status: draft
---

# All Outputs Committed Together

After the coach's meeting-processing skill finishes, all outputs (todos, specs, context) are committed in a single commit with clear categorization.

## Success Criteria

- Single git commit includes:
  - TODOS.md (if todos were found)
  - New spec files in specs/ (if features were discussed)
  - Updated CONTEXT.md (if decisions were made)
- Commit message format: `Process meeting: {date} - {brief summary}`
- Example: `Process meeting: 2025-02-12 - Job scraping alternatives and match score improvements`
- Each output type clearly labeled in commit message body:
  ```
  Todos:
  - Follow up with Kaiser team

  Specs created:
  - meeting-notes-to-specs

  Context updates:
  - Decided to use deep-link searches over scraping
  ```
- Commit happens only after user approves all outputs
