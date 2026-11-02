---
id: evidence-level-is-strongest-passing
parent: verify-command
created: 2026-11-02T21:30:00Z
priority: 1
status: not_started
depends-on: results-map-to-their-properties
---

# The Evidence Level of a Property Is the Strongest Level Among Its Passing Checks

## Description

The ladder is `example` < `property` < `contract` < `bounded` < `proof`. The level of a property is the strongest level of a marker whose test passed at HEAD. A marker that did not pass adds nothing.

## Success criteria

- A property with a passing `example` check and a passing `property` check has level `property`.
- A property with a passing `example` check and a failing `property` check has level `example`, and its truth is `failing`. Both values show (`example · failing`).
- A property with no passing check at HEAD has no level. The level is empty, not `example`.
- The level is calculated from markers and results only.
