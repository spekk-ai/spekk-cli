---
id: public-files-neutral
parent: sandbox-public-boundary
created: 2026-07-31T00:00:00Z
priority: 1
status: done
---

# Shipped Public Files Stay Inside the Boundary

No shipped public file — code, config, docs, specs, or release notes — names
the control host's implementation stack, its private repository name, a
specific private hostname presented as *the* host, or its internal admin URL
structure. See the parent spec for the full allowed/banned lists.

## Success Criteria

- A case-insensitive sweep of tracked files for the banned categories finds
  no hits, except: company attribution in `LICENSE` and site metadata, and
  example prompts that describe a hypothetical user's project.
- New text that mentions the control host uses the generic terms from the
  parent spec.

## Verification

There is no automated check: a grep list in a public repo would re-leak the
exact banned strings. Verification is a manual sweep at review time — run it
whenever a change touches text that mentions the control host, and after any
release-notes or spec addition about the sandbox subsystem.
