---
id: prompt-resolver-supports-per-agent-skills
parent: builder-skills-system
created: 2026-03-19T19:00:00Z
priority: 1
status: not_started
branch: feature/code-quality-qa
---

# PromptResolver supports per-agent skills directories with layered resolution

The `PromptResolver` in `src/cli/prompt-resolver.js` resolves skills directories per-agent using the same layered pattern as prompts (package → global → local), and the `createActivationMessage` includes the resolved skills paths in the activation message.

## Current State

- `createActivationMessage` hardcodes `skillsDir` to `specs/coach-skills-system` for ALL agents
- No layered resolution for skills — only package-level
- Builder and observer agents receive a coach skills directory reference they don't use

## Success Criteria

### Per-agent skills directory mapping

- `PromptResolver` has a skills directory mapping per agent:
  - `coach` → `specs/coach-skills-system/` (package level, already exists)
  - `builder` → `specs/builder-skills-system/` (package level, new)
  - `observer` → none (no skills system yet)
  - `reviewer` → none (uses gates, not skills)

### Layered skills resolution

- For each agent that has skills, resolve from three layers:
  1. **Package skills**: `<spekk-package>/specs/<agent>-skills-system/` (base, always present)
  2. **Global skills**: `~/.spekk/<agent>-skills/` (user's personal skills across projects)
  3. **Local skills**: `.spekk/<agent>-skills/` (project-specific skills)
- Local skills with the same filename override package skills
- All three paths are passed to the agent in the activation message so the agent knows where to look

### Updated activation message

- `createActivationMessage` outputs per-agent skills information:
  - Coach gets: `Skills directory: <package>/specs/coach-skills-system` (+ global/local if they exist)
  - Builder gets: `Skills directory: <package>/specs/builder-skills-system` (+ global/local if they exist)
  - Observer/reviewer get no skills directory line
- Only agents with registered skills directories get the `Skills directory:` line
- If global or local skills directories exist, they appear as additional lines:
  ```
  Skills directory: /opt/homebrew/lib/node_modules/@spekk/cli/specs/builder-skills-system
  Global skills: ~/.spekk/builder-skills/
  Local skills: .spekk/builder-skills/
  ```

### Testing

- Tests verify coach activation message includes coach skills dir (existing behavior preserved)
- Tests verify builder activation message includes builder skills dir (new behavior)
- Tests verify observer activation message does NOT include a skills directory
- Tests verify layered resolution: local skills dir is included when `.spekk/<agent>-skills/` exists
