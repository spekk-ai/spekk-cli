---
id: public-files-neutral
parent: sandbox-public-boundary
created: 2026-07-31T00:00:00Z
priority: 1
status: done
---

# Shipped Public Files Stay Inside the Boundary

No shipped public file — code, config, docs, specs, or release notes — names the control host's implementation stack, its private repository name, a specific private hostname presented as *the* host, or its internal admin URL structure. See the parent spec for the full allowed/banned lists.

## Success Criteria

- A case-insensitive sweep of tracked files for the banned categories finds no hits, except: company attribution in `LICENSE` and site metadata, and example prompts that describe a hypothetical user's project.
- New text that mentions the control host uses the generic terms from the parent spec.

## Verification

Review, informed by the agent prompts. There is no automated check, and a term denylist was tried and rejected — see the parent spec for why, so it is not rebuilt.

Run a sweep when a change adds prose about the control host, and when a release note or spec describes work done elsewhere. A review agent does this well: give it the list of private repositories and ask it to cross-reference them against the diff. It also catches material no denylist could, because it reads for meaning rather than matching strings.
