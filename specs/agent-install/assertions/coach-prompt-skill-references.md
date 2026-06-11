---
id: coach-prompt-skill-references
parent: agent-install
created: 2026-06-11T00:00:00Z
priority: 2
status: done
branch: feat/agent-install
---

# Coach Prompt References Skills via `spekk skill`, Not Repo Paths

The coach prompt instructs agents to fetch skill content with `spekk skill show coach <name>` instead of reading `specs/coach-skills-system/*.md` from disk, so the prompt works in any project — not just inside the spekk-cli repo.

## Success Criteria

- `specs/coach-agent/coach.prompt.md` tells the agent to discover skills with `spekk skill list coach` and load one with `spekk skill show coach <name>`.
- Reading skill files directly from `specs/coach-skills-system/` is mentioned only as a fallback for when the `spekk` CLI is unavailable (e.g., working inside this repo without the binary).
- No other instruction in the prompt depends on the spekk-cli repo's own directory layout being present in the user's project.

**Tests:** prompt content is verified by inspection; resolution behavior covered by `internal/cli/skill_test.go`.
