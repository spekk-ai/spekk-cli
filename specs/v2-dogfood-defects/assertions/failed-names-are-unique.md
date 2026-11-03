---
id: failed-names-are-unique
parent: v2-dogfood-defects
created: 2026-11-03T06:49:00Z
priority: 1
status: done
branch: fix/v2-dogfood-defects
depends-on: dirty-truth-is-labeled
---

# Failed Test Names Are Unique in the Report

## Success criteria

- A parent test and its failed subtest map to the same property, but the failing line names the test once.
- Distinct failed test names keep their first-seen order.
