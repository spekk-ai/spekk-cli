---
id: gitignore-neutral-reference-path
parent: sandbox-boundary-scrub
created: 2026-07-23T00:00:00Z
priority: 3
status: done
---

# .gitignore's Ignored Reference Path Is Neutrally Named

`.gitignore` (around line 84) ignored a reference directory whose name embeds
the control host's private repository name and ships in a public repo.

## Success Criteria

- `.gitignore` no longer contains the private repo name: a search for the
  private server repo's name in `.gitignore` returns nothing.
- The ignore rule is **renamed, not deleted** — a neutral pattern (e.g.
  `*-reference/` or `.reference/`) still ignores the local reference
  checkout developers keep in that directory.
- **Coordination cost (announce, do not do silently):** renaming the ignored
  path can force a local-workflow change. Any developer who keeps a working
  directory under the old name must stay covered, or must rename it to the new
  ignored path; otherwise their local reference checkout is no longer ignored
  and could be accidentally committed. Because this touches every contributor's
  local setup, the rename is coordinated and announced (e.g. in release notes
  or a team heads-up) rather than bundled silently into a code cycle — which is
  why it was queued at P3.
- **Resolution:** the rule is now the glob `*-reference/`. The glob also
  matches the old directory name, and a separate root rule already ignored
  that directory as well, so no contributor must rename anything.
