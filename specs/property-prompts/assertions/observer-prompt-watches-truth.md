---
id: observer-prompt-watches-truth
parent: property-prompts
created: 2026-11-02T21:30:00Z
priority: 2
status: not_started
---

# The Observer Prompt Treats a Property That Is Not True at HEAD as Drift

## Description

A stale property is worse than no property, because an agent trusts it. The observer is the agent that finds such drift. Its prompt must use the truth at HEAD and the evidence as inputs.

## Success criteria

- The observer prompt calls the unit a "property" and does not use the word "declarative".
- It tells the observer to read the truth at HEAD (`spekk list`, or `spekk query` on `verify_truth`) and to treat a `done` property that is `failing` or `unconfirmed` as a drift candidate.
- It tells the observer to check that a marked test checks the statement of its property, and to record weak evidence (a marker on a test that does not check the statement) as a finding.
- It tells the observer to record an anchor that is not at an enforcement point as a finding.
- Mutation testing is not in this prompt. It is later work.
- The observation output contract and the observation types do not change.
- `spekk prompt observer` prints the new text.
