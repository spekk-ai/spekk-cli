---
id: builder-loop-review-integration
parent: reviewer-agent
created: 2026-03-19T18:13:00Z
priority: 3
status: done
depends-on: review-cli-exists
branch: feature/code-quality-qa
---

# Builder loop supports optional --review flag

The builder loop (`src/loops/index.js`) supports an optional `--review` flag that runs `spekk review --no-llm` after each successful build and commit.

## Success Criteria

- `spekk loop builder --review` runs `spekk review --no-llm` after each build iteration completes and commits
- Review results are logged to the console (applicable gates, skipped gates, any findings)
- Review failures do NOT block the next build iteration (warnings only)
- Without `--review` flag, builder loop behavior is unchanged (no review step)
- `spekk loop --help` documents the `--review` flag
