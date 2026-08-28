---
id: covering-needs-the-owning-branch
parent: only-a-live-claim-covers
created: 2026-08-28T12:00:00Z
priority: 1
branch: fix/only-a-live-claim-covers
status: done
---

# An Observation Covers Only From Its Own Branch, and Only Until It Reaches Main

An observation speaks for a finding when the branch it is on is the branch named after it. Anywhere else it is a copy that a branch inherited, and a copy is not a claim.

## Success criteria

- `internal/observation/union.go` holds one predicate for "this observation is a live claim", and `FindCovering` and `Digest` both use it. Two surfaces that answer "which findings are live" must not answer it differently.
- The predicate is true only when both hold:
  - The ref the observation was read from is its own branch: the ref's logical branch name equals `BranchName(o.Slug)`. `refs/heads/observer/<slug>` and `refs/remotes/<remote>/observer/<slug>` both qualify; any other branch, and main or master, do not.
  - `u.OnMain(o.Slug)` is false. A slug on main is resolved history, so the claim has ended even while its branch survives.
- `isMainRef` is no longer consulted by `FindCovering`. Main is never `observer/<slug>`, so the owning-branch test already excludes it, and a second test for the same thing invites the two to disagree. `isMainRef` stays where `OnMain` needs it.
- `Digest` keeps its present output. It drops `!isMainRef(o.Ref)` for the shared predicate, and it still requires `status: open`, still excludes a slug on main, still shows each slug once, still ranks by `SortForDigest`, and still caps at `DigestCap`.
- A branch-local `status: resolved` is **not** part of the predicate. See the parent spec for why: the frontmatter is allowed to lag, and a status flipped while the remedy PR is still open would end a claim that is live.
- `status: dismissed` on the observation's own branch still covers, as it does today. A dismissal is a decision a person recorded, and the branch that carries it is still there.

**Note:** the merged-but-undeleted branch is settled by this assertion, not left open. Both facts are true at once — the branch is visible and the slug is on main — and main wins, so the finding is resolved. The next scan of the same drift is `clear` and `ResolveSlug` gives it a dated slug, so the new branch does not collide with the old one.

**Tests:** `internal/observation/` — a repository with one observation resolved on main and one unrelated open observer branch cut from that main: the inherited copy does not cover, and the open finding on its own branch still does; a branch that carries an observation named after a different branch does not cover; a branch merged to main but not deleted does not cover. `internal/observation/` — the digest over the same repository is unchanged.
