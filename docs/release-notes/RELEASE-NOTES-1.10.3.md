# Spekk CLI 1.10.3 — Differential Diagnosis Placement Fix

A follow-up to 1.10.2. The differential diagnosis protocol was in the coach prompt but didn't reliably fire — this release fixes where and how it's stated so it works across all three supported models (Opus, Sonnet, Haiku).

## What changed

The protocol section was moved from mid-prompt to the **end of the prompt**, and restructured so the ⛔ prohibition leads before any explanatory framing. Mid-prompt placement failed all three models in evals even with strong wording — the models entered explanation mode before they reached the prohibition.

**Eval COACH-01:** 0/FAIL across 3 models (1.10.2) → 2/PASS across 3 models (1.10.3), confirmed by an LLM judge.

This resolves the failure mode where the coach would commit to a hypothesis without first asking the question that determines the answer.

## Upgrade

```bash
spekk update
```
