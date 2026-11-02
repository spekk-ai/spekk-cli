---
id: property-prompts
created: 2026-11-02T21:30:00Z
priority: 2
---

# Property Prompts - the Agents Write, Build and Watch Properties

## Problem

The coach, builder and observer prompts teach the v1 unit: an assertion that "declares state". v2 makes the unit a property with evidence. If the prompts keep the old terms, the agents keep writing steps, the builder writes tests that name no property, and the observer does not see a stale property.

The prompts are `specs/coach-agent/coach.prompt.md`, `specs/builder-agent/builder.prompt.md` and `specs/observer-agent/observer.prompt.md`. The layered prompt system (`layered-prompt-system`) does not change.

## Decisions

- "Property" is the unit, and "invariant" is one kind of property. The word "declarative" goes out of the coach prompt and the CLI output (William, 2026-11-01).
- Write properties, not steps. Steps do not compress, and they go stale first.
- Point to the code where v1 specs describe it ("Implementation Notes").
- When the coach sees an invariant, it proposes an enabling property, for example "only the ledger writes `balance`". This makes one enforcement point that one anchor can mark.

## Open choices

- **Scope of "CLI output".** Today no command output uses "declarative". The coach prompt is the only place that applies the word to specs. The observer prompt also says that observer state lives "declaratively" in the repo. That is not about specs, but the observer prompt drops the word too, so that one rule covers all three prompts.
- **The eval.** COACH-06 in spekk-evals uses the old word. That repository is out of scope here.

## Assertions

See `assertions/` for what must be true.
