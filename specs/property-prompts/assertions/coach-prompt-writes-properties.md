---
id: coach-prompt-writes-properties
parent: property-prompts
created: 2026-11-02T21:30:00Z
priority: 2
status: not_started
---

# The Coach Prompt Teaches Properties, Not Steps

## Description

The coach writes the specs. Its prompt must make it write properties with their intent, in the v2 format, and must not teach the v1 framing.

## Success criteria

- The coach prompt calls the unit a "property" and calls "invariant" one kind of property. The word "declarative" does not occur in it.
- It tells the coach to write a property as a rule that must stay true, not as steps, and gives the reason: steps do not compress, and they go stale first.
- It shows the property format of `property-format`: `kind`, `## Terms`, `## Decisions`, `## Rationale`, and front-matter `anchors` for a property with no checks. It names the five kinds.
- It tells the coach to point to the code (a path, a function) where a v1 spec describes the code in "Implementation Notes".
- It tells the coach, for each invariant, to propose an enabling property that makes one enforcement point (for example "only the ledger writes `balance`").
- It says that a v1 assertion with none of the new parts is still valid, and that the coach adds the new parts where the value is high.
- The status values and the required fields in the prompt stay the v1 values.
- `spekk prompt coach` prints the new text.
