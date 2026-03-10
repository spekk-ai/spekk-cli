---
id: publish-workflow-uses-self-hosted-runner
parent: self-hosted-gh-runners
created: 2026-03-10T00:00:00Z
priority: 1
status: not_started
branch: feature/self-hosted-runners
---

# Publish Workflow Uses Self-Hosted Runner

`.github/workflows/publish.yml` runs on the self-hosted DO droplet runner inside a Node container.

## Success Criteria

- `runs-on` is `[self-hosted, linux]` (not `ubuntu-latest`)
- `container.image` is `node:20-slim`
- All existing publish steps are preserved (npm ci, npm test, npm pack, GemFury upload)
