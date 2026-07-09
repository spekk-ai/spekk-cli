---
id: post-build-skills
parent: builder-loop
created: 2026-07-09T16:00:00Z
priority: 2
status: in_progress
depends-on: exit-on-complete
branch: feature/advanced-loop
locked-by: builder-Paris-MacBook-Pro-2.local-47012-1783627340
---

# Post-Build Skills Pipeline

The builder loop accepts skill names as positional arguments and runs them sequentially after all assertions complete.

## Success Criteria

- `spekk loop builder e2e-testing-skill api-audit-skill` stores the skill names for post-build execution
- Skills only run when the loop completed at least one assertion (no skills on "nothing to do")
- Each skill is launched as a separate builder agent invocation (reuses existing `spekk builder <skill>` machinery)
- A checklist is displayed showing skill progress: `[x] e2e-testing-skill`, `[ ] api-audit-skill`
- If a skill fails or times out, the loop logs it and continues to the next skill
- After all skills run, the loop prints a summary: `"Post-build skills: 2/2 completed"` and exits
