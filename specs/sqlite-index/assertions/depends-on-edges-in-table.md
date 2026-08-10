---
id: depends-on-edges-in-table
parent: sqlite-index
created: 2026-07-12T22:00:00Z
priority: 1
status: done
depends-on: index-command-builds-db
---

# All `depends-on` Frontmatter Fields Are Stored as Edges in `depends_on` Table

## Description

When an assertion's frontmatter contains a `depends-on` field, `spekk index`
inserts a row into the `depends_on` table linking the assertion to the target.
This enables constant-time reverse dependency queries via SQL JOIN.

## Success Criteria

- Given an assertion with `depends-on: foo-assertion` in its frontmatter,
  after `spekk index` the `depends_on` table contains a row
  `(assertion_id='<this-assertion-id>', depends_on_id='foo-assertion')`.
- An assertion with no `depends-on` frontmatter field produces no row in
  `depends_on` (the table is sparse).
- `spekk query "SELECT assertion_id FROM depends_on WHERE depends_on_id = 'X'"`
  returns all assertions that depend on X (reverse dependency lookup).
- A unit test or integration test verifies the edge count equals the number of
  assertions in the fixture that have `depends-on` set.
