---
id: observation-tables-schema
parent: observation-index
created: 2026-07-26T12:00:00Z
priority: 1
status: done
depends-on: observation-file-format
---

# The Index Contains Observation Tables Mirroring the Frontmatter Schema

## Description

`spekk index` populates observation tables (`observations` and
`observation_files`, or an equivalent normalized shape) in `.spekk/index.db`,
carrying every frontmatter field from the observation format plus the ref the
row was read from.

## Success Criteria

- After `spekk index` in a repo containing observations, the index contains:
  - one observations row per (observation, ref) pair, with columns covering
    `slug`, source ref, `type`, `severity`, `status`, `created`, `announced`
    (NULL when the frontmatter field is absent), `pr`, and source file path
  - one evidence row per `affected` path per observation, keyed to the same
    (slug, ref)
- `spekk query "SELECT slug, severity, status FROM observations"` returns the
  indexed observations; the evidence table is joinable on the (slug, ref) key.
- An observation file that fails the evidence gate (missing/empty `affected`)
  or has invalid frontmatter is not silently indexed as valid: it is either
  skipped with a warning naming the file, or indexed with a marker that
  downstream consumers (announce) treat as ineligible. It must never appear
  as an announceable open observation.
- `spekk index --force` drops and rebuilds these tables along with the
  existing ones; the schema-version mechanism from `specs/sqlite-index/`
  covers the new tables (schema change → version bump → rebuild).

**Note:** `announced` must be distinguishable as *absent* (SQL NULL), not
empty string — the announce subcommand's eligibility test is `announced IS
NULL`, and conflating the two re-creates the ambiguity this design removes.

**Tests:** internal/index/observation_test.go (TestObservationTablesPopulated,
TestObservationInvalidFileSkippedWithWarning,
TestObservationSameSlugMultipleRefs)
