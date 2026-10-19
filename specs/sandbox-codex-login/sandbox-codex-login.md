---
id: sandbox-codex-login
created: 2026-10-19T00:00:00Z
priority: 1
---

# Sandbox provisioning includes a Codex CLI and ChatGPT login

## Problem

A subscription sandbox currently has Claude Code only. An operator cannot select Codex for a headless handoff using the same sandbox setup.

## Design

Install a pinned Codex CLI beside Claude Code on both cloud and existing machine paths. For subscription mode, require a file-backed ChatGPT login cache before creating a machine. Copy it over SSH stdin into a private file owned by `agent`. Keep Claude's token path working. Do not use an API key or a local proxy.

A copied ChatGPT refresh token cannot be treated as a durable, concurrent credential. Each machine persists its refreshed file, but independent copies can diverge. A multi-machine refresh requires separate sign-ins or another credential type, and needs a real-cloud check.

## Rejected

- OpenAI API keys, which use separate billing.
- Unpinned CLI installs, which can change between machines.
- Shell arguments for transporting credentials.
