---
id: test-workflow-uses-self-hosted-runner
parent: self-hosted-gh-runners
created: 2026-03-10T00:00:00Z
priority: 1
status: in_progress
locked-by: builder-warespace-i7-205285-1773184722
branch: feature/self-hosted-runners
---

# Test Workflow Uses Self-Hosted Runner

`.github/workflows/test.yml` runs on the self-hosted DO droplet runner inside a Node container.

## Success Criteria

- `runs-on` is `[self-hosted, linux]` (not `ubuntu-latest`)
- `container.image` is `node:20-slim`
- All existing test steps are preserved
