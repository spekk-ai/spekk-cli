---
id: show-output-stable-with-custom-fields
parent: show-custom-fields
created: 2026-11-19T14:55:40Z
priority: 2
status: not_started
depends-on: show-data-carries-custom-fields
branch: feature/show-custom-fields
---

# Show Output Is Deterministic With Custom Fields

## Success Criteria

- Two runs of `spekk show --json` over the same tree, which includes items with several custom keys, give byte-identical stdout.
- Two runs of `spekk show` over the same tree write a byte-identical `.spekk/index.html`.
- Within `customFields`, keys appear in alphabetical (byte) order. Values keep the order they have in the frontmatter, with the duplicates removed.
- A test runs the generation at least 20 times on one fixture tree and asserts one distinct output, so that Go's random map iteration order cannot pass by chance.
