---
id: go-test-markers-follow-go-rule
parent: v2-dogfood-defects
created: 2026-11-03T06:49:00Z
priority: 1
status: done
branch: fix/v2-dogfood-defects
depends-on: junit-input-is-valid
---

# Go Test Markers Follow the Go Name Rule

## Success criteria

- A marker above `func Test(t *testing.T)` or `func Test_x(t *testing.T)` gives evidence and is not an anchor.
- A function whose name starts with `Test` followed by a lower-case letter remains outside the test marker rule.
