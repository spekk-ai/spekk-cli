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

`scripts/check-private-terms.sh`, in CI on every push.

This assertion previously recorded that no automated check was possible, because a grep list committed to a public repo would re-leak the exact strings it banned. That reasoning is right and the conclusion does not follow: the list does not have to be committed. The terms come from the `SPEKK_PRIVATE_TERMS` secret, so CI can match against them while the public tree never holds them.

A manual sweep is still worth doing when a change adds prose about the control host, because the script only catches terms someone has already thought to add. It is no longer the only defence.
