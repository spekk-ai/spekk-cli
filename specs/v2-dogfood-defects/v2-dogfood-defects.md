---
id: v2-dogfood-defects
created: 2026-11-03T06:49:00Z
priority: 1
---

# V2 Dogfood Defects

## Problem

Four verification and display errors give incomplete or incorrect evidence for a property.

## Decision

Keep the current truth model and output shapes. Report the dirty state with the run, reject files that are not JUnit XML, and use Go's test-name rule for markers.
