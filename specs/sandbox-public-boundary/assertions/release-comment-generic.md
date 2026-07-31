---
id: release-comment-generic
parent: sandbox-public-boundary
created: 2026-07-23T00:00:00Z
priority: 2
status: done
---

# release.go Comment Names No Private Repo

`internal/sandbox/release.go` fetches release artifacts from the public repo
named in its `releaseRepo` constant (`spekk-ai/spekk-cli`). A doc comment
described the source as a GitHub release of the private server repo, named
directly, which both named a private repo and contradicted the constant.

## Success Criteria

- The comment on `fetchReleaseArtifacts` (and any other comment in the file)
  describes the artifact source generically — e.g. "from the project's GitHub
  release" or "from the release named by `releaseRepo`" — with no private repo
  name.
- A search for the private server repo's name in `internal/sandbox/release.go`
  returns nothing.
- The `releaseRepo` constant value is unchanged (it already points at the public
  repo); only comment text changes. No behavior changes.
