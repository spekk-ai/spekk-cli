---
id: results-map-to-their-properties
parent: verify-command
created: 2026-11-02T21:30:00Z
priority: 1
status: not_started
depends-on: verify-runs-configured-commands
---

# Each JUnit Result Maps to the Markers of the Test That Produced It, and to No Other

## Description

A JUnit `testcase` maps to a property only through the marker on the test that produced it. A test with the same name in another package or module must never give evidence for a property that it does not check.

## Success criteria

- A Go result maps to the markers of the function with the same name in the package that the testcase `classname` names. A subtest (`TestX/case`) maps to the markers of `TestX`.
- A pytest result maps to the markers of the function that its `classname` (the dotted module and class path) and `name` name. A parametrized case (`test_x[a-1]`) maps to the markers of `test_x`.
- Two marked tests with the same function name in two packages or modules each get only their own results.
- A result whose test has no marker gives no evidence. It does not change the truth of any property.
- A result is `passed`, `failed` (a `failure` element), `error` (an `error` element) or `skipped` (a `skipped` element).
- A marked test with no result in the run is listed by name on stderr, as `no result: <file>:<test>`. It does not stop the run.
- **Tests:** two packages with the same test name; a Go subtest; a pytest parametrized case; an unmarked failing test changes no property.
