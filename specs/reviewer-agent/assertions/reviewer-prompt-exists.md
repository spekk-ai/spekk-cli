---
id: reviewer-prompt-exists
parent: reviewer-agent
created: 2026-03-19T18:11:00Z
priority: 2
status: done
depends-on: review-cli-exists
branch: feature/code-quality-qa
---

# Reviewer agent prompt exists and is registered

A reviewer agent prompt file exists and is registered in the PromptResolver, following the same pattern as coach/builder/observer agents.

## Success Criteria

- Prompt file exists at `specs/reviewer-agent/reviewer.prompt.md`
- Prompt defines the reviewer agent's role: read-only quality validation through gates
- Prompt explains the two-phase evaluation: deterministic preconditions already passed, reviewer handles LLM judgment + workflow execution
- Prompt instructs the reviewer to:
  1. Read the gate's `## LLM Judgment` section and decide if the gate should run
  2. If yes, execute the gate's `## Workflow` section
  3. Report results with severity level from `## On Failure`
  4. Never modify implementation code — only report findings
- `src/cli/prompt-resolver.js` includes `{ name: 'reviewer', path: '...reviewer.prompt.md' }` in `promptFiles`
- `spekk review` (without `--no-llm`) launches the reviewer agent via Claude Code, same pattern as observer CLI
- `package.json` `files` array includes `specs/reviewer-agent/` so the prompt ships with the package
