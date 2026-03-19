---
id: review-cli-exists
parent: reviewer-agent
created: 2026-03-19T18:10:00Z
priority: 1
status: done
depends-on: gate-engine-evaluates-preconditions
branch: feature/code-quality-qa
---

# CLI command `spekk review` exists with flags

The `spekk review` command is registered in the CLI and supports the core flags.

## Success Criteria

- `bin/spekk.js` has a `case 'review'` that routes to `src/reviewer/cli.js`
- `src/reviewer/cli.js` exists and handles the following flags:
  - `--list`: shows applicable/skipped gates with deterministic reasons only (no LLM)
  - `--dry-run`: full evaluation including LLM judgment but doesn't execute gate workflows
  - `--gate <id>`: evaluate and run a specific gate only
  - `--force <id>`: force-run a gate, skipping all preconditions and LLM judgment
  - `--skip <id>`: force-skip a gate
  - `--no-llm`: deterministic precondition checks only, no LLM judgment phase
  - `--tags <tag>`: filter gates by tag
  - `--help`: shows usage information
- `--list` output shows two groups: "Applicable" (gates that passed preconditions) and "Skipped" (gates that failed, with reason)
- `package.json` has `"review": "node bin/spekk.js review"` script
- Help text is registered in the main `spekk --help` output
