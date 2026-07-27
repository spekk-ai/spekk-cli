---
id: suppression-requires-review
parent: observer-dont-flag
created: 2026-07-26T12:00:00Z
priority: 2
status: done
depends-on: dont-flag-file-schema
---

# Permanent Dismissal of a Finding Is a Reviewed PR Adding a Dont-Flag Entry

## Description

The only sanctioned path to permanently dismissing a class of drift is a
small PR against main that adds an entry to `.spekk/dont-flag.yaml`. This
keeps every suppression human-gated: it always carries a reviewer, a stated
reason, and a named author.

## Success Criteria

- The documented workflow for "the observer keeps finding X and we've decided
  X is fine" is: open a PR adding a `dont-flag.yaml` entry with `match`,
  `reason`, and `by` filled in; merge after review. This workflow is stated in
  the parent spec and in the observer prompt.
- The observer itself never writes to `.spekk/dont-flag.yaml` — not on main,
  not on observer branches. It is a human-authored file; the observer only
  reads it.
- Observer-generated PR bodies that offer a dismissal path point to this
  mechanism for permanent dismissal (see
  `specs/observation-lifecycle/assertions/pr-body-conventions.md`), and
  distinguish it from branch deletion (which invites re-flagging).
- Nothing in the tooling provides a way to suppress drift that bypasses the
  file — there is no CLI flag, environment variable, or prompt instruction
  that silently mutes a class of findings without a committed, attributed
  entry.

**Note:** enforcement of "merge after review" is a repo-policy concern
(branch protection), not something spekk can guarantee; the assertion's scope
is that the *workflow and tooling* offer no unreviewed alternative.
