---
id: sandbox-boundary-scrub
created: 2026-07-23T00:00:00Z
priority: 1
---

# Sandbox Boundary Scrub

## Overview

This repository is public. The orchestration server it talks to (the "control
host") is a separate, private application. Several shipped files in this repo
leak details of that private side — its implementation stack, a private repo
name, and a specific admin URL shape — and one file puts the agent's auth token
in a place it should not be. This spec scrubs the shipped code and config so a
public reader learns nothing about the control host's internals, and hardens
how the agent presents its auth token.

These files are being touched by the conversation-open work anyway, so the
cleanup rides along in the same cycle.

## What the boundary allows

- Referring to the server only generically: "control host", "orchestration
  server", or a neutral host placeholder.
- Naming a chat surface (e.g. Slack) only as **one example** of what a control
  host might bridge to — never as the implementation.

## What must not appear in shipped public files

- The control host's implementation stack.
- The control host's private repository name.
- A specific private hostname presented as *the* host (neutral placeholders are
  fine).
- Internal admin URL structure of the control host.

## In scope (each an assertion below)

Concrete leak sites found by sweeping `*.go` / `*.yaml` in shipped code:

- `internal/sandbox/cloud-init.yaml` — a comment naming the stack ("WebSocket
  connection to Django") and a specific host example (`app.spekk.ai`).
- `internal/sandbox/release.go` — a comment sourcing artifacts from a named
  private repo ("from a spekk-app GitHub release"); the code constant already
  points at the public repo, so the comment is both a leak and stale.
- `internal/sandbox/commands.go` — user-facing CLI output that names the stack
  and the control host's admin URL ("Add this agent in Django admin at
  `https://%s/staff/agent/agent/add/`").
- `cmd/sandbox/client.go` — the agent auth token is embedded in the WebSocket
  URL path; move it to an `Authorization` header (hardening, coordinated with
  the control host).

## Out of scope for this cycle (flagged, not fixed here)

The sweep also found these public-file mentions; they are deliberately left out
so this cycle stays lean and touches only the shipped sandbox code:

- `docs/release-notes/RELEASE-NOTES-1.3.0.md` names the stack and presents one
  chat surface as *the* integration. Release notes are published history;
  editing them is a separate decision.
- `specs/sandbox-go-release/` (spec text and an assertion) names the private
  repo and is also stale (it still describes the pre-Go `*.js` layout). That is
  spec drift for the observer/coach to reconcile, not shipped code.
- `.gitignore` ignores a `spekk-app-reference/` path, which names the private
  repo. Renaming it forces a local-workflow change, so it is a separate call.
- Example prompt snippets in `README.md`, `docs/configuration.md`, and the
  `layered-prompt-system` spec say things like "a Django/HTMX project" — these
  describe a hypothetical *user's* project, not the control host, so they are
  not leaks and are intentionally left alone.

## Assertions

1. `cloud-init-neutral-comments` — scrub cloud-init.yaml comments
2. `release-comment-generic` — scrub the release.go comment
3. `cli-output-no-stack` — scrub the sandbox CLI's "add this agent" output
4. `ws-auth-header` — move the auth token to an Authorization header
