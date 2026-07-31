---
id: sandbox-public-boundary
created: 2026-07-23T00:00:00Z
priority: 1
---

# Sandbox Public Boundary

## Overview

This repository is public. The orchestration server it talks to (the "control
host") is a separate, private application. A boundary holds between the two:
a public reader of this repo learns nothing about the control host's
internals.

This is a standing condition, not a one-time cleanup. Violations have
recurred: an initial sweep cleaned the shipped code, and new leaks appeared
in later commits within days. The history of the individual cleanups lives in
git history and in the PRs that closed them.

## The boundary

Shipped public files (code, config, docs, specs, release notes) must not
name:

- The control host's implementation stack.
- The control host's private repository name.
- A specific private hostname presented as *the* host (neutral placeholders
  are fine).
- Internal admin URL structure of the control host.

Allowed:

- Referring to the server generically: "control host", "orchestration
  server", or a neutral host placeholder.
- Naming a chat surface (e.g. Slack) as **one example** of what a control
  host might bridge to — never as the implementation.
- Example prompts that name popular frameworks to describe a hypothetical
  *user's* project.

The same rule is stated for contributors in `CONTRIBUTING.md`.

## Assertions

1. `public-files-neutral` — shipped public files stay inside the boundary
   (done; re-verify on review whenever a file mentions the control host)
