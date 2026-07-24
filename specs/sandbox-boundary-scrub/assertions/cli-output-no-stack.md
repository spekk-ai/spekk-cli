---
id: cli-output-no-stack
parent: sandbox-boundary-scrub
created: 2026-07-23T00:00:00Z
priority: 2
status: not_started
---

# Sandbox CLI Output Names No Stack or Internal Admin URL

After `spekk sandbox create`, `internal/sandbox/commands.go` prints next-step
guidance telling the operator to "Add this agent in Django admin at
`https://%s/staff/agent/agent/add/`". This names the control host's stack and
exposes its internal admin URL structure in output shipped from a public repo.

## Success Criteria

- The printed guidance no longer names the stack. The word "Django" does not
  appear in the message; it refers to the destination generically (e.g. "the
  control host admin").
- The message still tells the operator that they need to register the agent
  (name, sandbox id, auth token) so the next step is not lost.
- A case-insensitive search for `django` in `internal/sandbox/commands.go`
  returns nothing.
- **Note:** the hard-coded internal path `/staff/agent/agent/add/` also reveals
  control-host internals. Prefer replacing it with the host root
  (`https://%s/`) or dropping the path; if it is kept, that is a conscious
  decision to surface, not an oversight. The stack name must go regardless.
