# Spekk CLI 1.10.2 — Differential Diagnosis Protocol

This release adds a diagnostic protocol to the coach agent. There are no CLI or binary changes — the update is entirely in the embedded coach prompt.

## Differential diagnosis for the coach

The coach's default behavior is to propose fast and let you react. That instinct is wrong for one class of question: *"why does X work for A but not B?"* — where jumping to a hypothesis before understanding the variables leads to confident, wrong answers.

This release adds a **differential diagnosis** protocol. When you ask why one entity has an outcome another doesn't — "why did this stop working?", "why does ingredient A show the flag but ingredient B doesn't?" — the coach now suspends its propose-fast rule and instead:

1. Enumerates the independent variables that could explain the difference
2. States which ones the question already rules out
3. Asks about the remaining open variables
4. *Only then* names a hypothesis

It also nudges the coach to consider whether a recurring root cause points to a missing spec.

The protocol was derived from a real eval failure where the coach committed to a root cause without first asking which code path was involved — the correct answer depended on knowing that detail up front. The prompt patch fixed the failure across all follow-up runs.
