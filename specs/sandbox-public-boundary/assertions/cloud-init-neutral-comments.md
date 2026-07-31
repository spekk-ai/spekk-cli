---
id: cloud-init-neutral-comments
parent: sandbox-public-boundary
created: 2026-07-23T00:00:00Z
priority: 1
status: done
---

# cloud-init.yaml Comments Are Stack- and Host-Neutral

`internal/sandbox/cloud-init.yaml` is shipped as droplet user-data from a public
repo. Its comments must not name the control host's implementation stack or a
specific private host.

## Success Criteria

- The comment on the `SPEKK_AGENT_TOKEN` line no longer names the stack. The
  phrase that named the stack ("WebSocket connection to ...") is replaced with
  a neutral description (e.g. "WebSocket connection to the control host").
- The `SPEKK_HOST` comment no longer uses a real private host as its example.
  The real hostname is replaced with a neutral placeholder (e.g.
  `your-control-host.example`). The comment still conveys that the value is the
  host the agent connects to.
- A case-insensitive search for the control host's stack name in
  `internal/sandbox/cloud-init.yaml` returns nothing.
- The file remains valid cloud-config YAML and the meaning of each comment
  (what the operator must fill in) is preserved.
