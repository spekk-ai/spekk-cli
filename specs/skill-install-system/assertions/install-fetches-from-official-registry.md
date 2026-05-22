---
id: install-fetches-from-official-registry
parent: skill-install-system
created: 2026-05-22T12:00:00Z
priority: 1
status: done
depends-on: install-command-parses-args
branch: feature/skill-install-system
---

**Tests:** internal/install/fetch_test.go

# Install Fetches from Official Registry by Default

## Description

When `--source` is not passed, the skill is fetched from `https://raw.githubusercontent.com/spekk-ai/spekk-skills/main/<agent>/<skill>.md`. Both the base URL and the API URL used for `--list` are overridable via env vars so users can point at forks/mirrors and tests can substitute an `httptest` server.

## Success Criteria

- Default raw base is `https://raw.githubusercontent.com/spekk-ai/spekk-skills/main`
- Default API base is `https://api.github.com/repos/spekk-ai/spekk-skills/contents`
- `SPEKK_SKILLS_RAW_BASE` env var overrides the raw base (trailing slash trimmed)
- `SPEKK_SKILLS_API_BASE` env var overrides the API base (trailing slash trimmed)
- The constructed URL for `spekk install coach foo` is `<raw-base>/coach/foo.md`
- HTTP requests set `User-Agent: spekk-cli`
- HTTP requests have a 30-second timeout
- A 404 response surfaces a clear "not found" error that names the URL attempted
- Any other non-2xx status surfaces a "http <code>: <url>" error
- The fetched body is written verbatim to the target path (no transformation)
