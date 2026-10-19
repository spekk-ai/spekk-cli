---
id: agent-attribution-setup
parent: sandbox-attribution
created: 2026-10-19T00:00:00Z
priority: 1
status: done
branch: feat/sandbox-attribution
---

# Sandbox setup applies the agent attribution policy

## Success Criteria

- Both cloud-init and the existing-machine setup give the agent `~/.local/bin/assisted-by` with the same bytes as the shared script in this repository.
- Cloud-init gets both setup files from one fixed revision that exists before this branch merges. The existing-machine instructions supply the setup script and both files from one checkout.
- The agent's `~/.claude/settings.json` has `attribution.commit` and `attribution.pr` set to empty strings. Other settings stay intact, including keys in `attribution`.
- The agent's global Git config sets `trailer.assisted-by.ifExists` to `addIfDifferent`.
- Repeating setup gives the same result. Invalid settings stop setup with an error.
- A test runs the setup against a temporary home with existing settings and verifies these results.

**Test:** `scripts/test_configure_agent_attribution.py`. The test runs the setup in a temporary home and checks a second run and invalid settings. `cloud-init schema` checks the cloud-init path.
