---
id: sandbox-codex-login
created: 2026-10-19T00:00:00Z
priority: 1
---

# Sandbox provisioning includes a Codex CLI and ChatGPT login

## Problem

A subscription sandbox currently has Claude Code only. An operator cannot select Codex for a headless handoff using the same sandbox setup.

## Design

Install a pinned Codex CLI beside Claude Code on both cloud and existing machine paths. For subscription mode, allow an optional file-backed ChatGPT login cache. Copy it over SSH stdin into a private file owned by `agent`. Keep Claude's token path working. Do not use an API key or a local proxy.

One ChatGPT login supports one active Codex machine at a time. If a caller seeds several machines from one file, the second refresh invalidates the other copies. The CLI does not lock machines or copy refreshed caches back. The caller must serialize use and copy back the latest cache.

## Rejected

- OpenAI API keys, which use separate billing.
- Unpinned CLI installs, which can change between machines.
- Shell arguments for transporting credentials.
