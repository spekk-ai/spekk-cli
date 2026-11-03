---
id: truth-at-head-follows-the-evidence
parent: verify-command
created: 2026-11-02T21:30:00Z
priority: 1
status: done
branch: feat/v2-verify-command
depends-on: results-map-to-their-properties
---

# The Truth of Each Property at HEAD Has Exactly One Value, From Its Evidence

## Description

Each property has one truth value at HEAD: `verified`, `failing`, `unchecked` or `unconfirmed`. The value comes only from recorded results for the HEAD commit, the markers, the anchors and git history. It never comes from a field in the property file.

## Success criteria

- A property with at least one marker:
  - is `failing` when one of its results in the latest run for HEAD is `failed` or `error`;
  - else is `verified` when at least one of its results in that run is `passed`;
  - else is `unchecked` (no run for HEAD, or only `skipped` results, or no results).
- A property with no marker:
  - is `unconfirmed` when a commit after the last commit that changed its assertion file changed a file that holds one of its anchors;
  - else is `unchecked`. A prose property with no anchors is always `unchecked`.
- Outside a git work tree, every property is `unchecked`.
- A new commit has no results until `spekk verify` runs on it. Thus after a commit, a property that was `verified` shows `unchecked` until the next run.
- Workflow `status` does not change the truth. A `not_started` property with passing checks is `verified`.
- A front-matter key such as `truth:` or `level:` is a custom field. It has no effect on the truth or the level.
- **Tests:** each of the four values; a failure and a pass in one run give `failing`; a run on the previous commit does not count for HEAD; an anchored file change after the last change of the property file gives `unconfirmed`, and a new commit to the property file returns it to `unchecked`.
