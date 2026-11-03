---
id: junit-input-is-valid
parent: v2-dogfood-defects
created: 2026-11-03T06:49:00Z
priority: 1
status: done
branch: fix/v2-dogfood-defects
depends-on: failed-names-are-unique
---

# JUnit Input Is Valid Before It Gives Evidence

## Success criteria

- An empty file or an XML file whose root is not `testsuite` or `testsuites` gives `<path>: not JUnit XML` on stderr and makes verify exit 1.
- A valid JUnit root with zero test cases stays valid.
