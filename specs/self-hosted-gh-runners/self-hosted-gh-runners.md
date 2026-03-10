---
id: self-hosted-gh-runners
created: 2026-03-10T00:00:00Z
priority: 1
---

# Self-Hosted GitHub Actions Runners

GitHub Actions workflows use self-hosted runners on the DO droplet instead of GitHub-hosted runners, eliminating GitHub Actions minute consumption.

## Context

The spekk-ai/spekk-app repo already has self-hosted runners configured on a DigitalOcean droplet. This spec mirrors that setup for the spekk-cli-runners repo.

## Manual Prerequisite (ops step, not builder work)

Before workflows will succeed, register a new runner on the DO droplet for this repo:

1. Go to GitHub → spekk-ai/spekk-cli-runners → Settings → Actions → Runners → New self-hosted runner
2. Follow the registration steps on the droplet
3. The runner will pick up jobs tagged `[self-hosted, linux]`

## Done When

Both workflow files use `runs-on: [self-hosted, linux]` with a containerized Node environment, matching the pattern in spekk-app.
