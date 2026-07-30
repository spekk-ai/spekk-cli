---
id: gitignore-neutral-reference-path
parent: sandbox-boundary-scrub
created: 2026-07-23T00:00:00Z
priority: 3
status: not_started
---

# .gitignore's Ignored Reference Path Is Neutrally Named

`.gitignore` (around line 84) ignores `spekk-app-reference/`, whose name embeds
the control host's private repository name and ships in a public repo.

## Success Criteria

- `.gitignore` no longer contains the private repo name: a search for
  `spekk-app` in `.gitignore` returns nothing.
- The ignore rule is **renamed, not deleted** — a neutral path (e.g.
  `control-host-reference/` or `.reference/`) still ignores the local reference
  checkout developers keep in that directory.
- **Coordination cost (announce, do not do silently):** renaming the ignored
  path forces a local-workflow change. Any developer who keeps a working
  directory named `spekk-app-reference/` must rename it to the new ignored path;
  until they do, their local reference checkout is no longer ignored and could
  be accidentally committed. Because this touches every contributor's local
  setup, the rename is coordinated and announced (e.g. in release notes or a
  team heads-up) rather than bundled silently into a code cycle — which is why
  it is queued at P3.
