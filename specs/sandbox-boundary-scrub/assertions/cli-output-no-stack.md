---
id: cli-output-no-stack
parent: sandbox-boundary-scrub
created: 2026-07-23T00:00:00Z
priority: 2
status: done
---

# Sandbox CLI Output Names No Stack or Internal Admin URL

After `spekk sandbox create`, `internal/sandbox/commands.go` printed next-step
guidance that told the operator to add the agent on the control host's admin
page, naming the control host's stack and giving the admin page's internal URL
path. This exposed the control host's stack and its internal admin URL
structure in output shipped from a public repo.

## Success Criteria

- The printed guidance no longer names the stack. The stack name does not
  appear in the message; it refers to the destination generically (e.g. "the
  control host admin").
- The internal admin URL path is **removed** from the output — not merely the
  stack name. It leaks the private app's URL structure, so it must go. Replace
  it with the host root (`https://%s/`) or drop the URL from the message
  entirely; the output must not contain the admin path prefix (or any deeper
  admin path).
- The message still tells the operator that they need to register the agent
  (name, sandbox id, auth token) so the next step is not lost.
- A case-insensitive search for the control host's stack name in
  `internal/sandbox/commands.go` returns nothing, and a search for the admin
  path prefix in the "add this agent" output returns nothing.
