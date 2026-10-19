---
id: sandbox-attribution
created: 2026-10-19T00:00:00Z
priority: 1
---

# Sandbox agents use one attribution policy

## Problem

Claude Code can add AI co-author text to commits and AI credit to pull request bodies. The sandbox setup does not set its attribution policy. The agent also lacks the shared `assisted-by` tool and the Git trailer rule.

## Design

Set Claude Code's commit and pull request attribution values to empty strings while keeping other settings. Install the shared `assisted-by` script for the agent and set Git to add a different Assisted-by trailer. Apply this setup in cloud-init and the setup for an existing machine.

Codex 0.163.0 reads its attribution policy from the signed-in account. Its local configuration has no attribution switch. The account setting must be off for Codex to omit its own credit lines.
