# Spekk CLI 1.13.0 — The `prune` Observer Skill

The dev loop is additive by construction — coach adds assertions, builders add code — and nothing is charged with asking *what should be removed*. This release adds the counter-pressure.

## `spekk observer prune`

A new opt-in observer skill that surfaces **genuinely-unused code and design-level redundancy** as observations for human review, across four evidence-backed lenses: unused code (no caller / no test / no reference), duplication, over-abstraction, and dead configuration.

```bash
spekk observer prune
```

It is **recommend-only** — the observer's read-only contract holds, so it writes observations and never touches code — and precision-biased (it omits anything with a plausible reason to keep: public API, unresolved dynamic callers, recent additions). Crucially, deletion keys on genuine *disuse*, **never on the absence of a spec**: specs are progressive, so most code legitimately has no owning assertion, and that is normal.

## `coverage-gap` reframed

The existing `coverage-gap` skill was realigned to the same progressive-spec philosophy. It no longer scores a missing spec as severity or implies "no spec → delete it"; it now surfaces **optional documentation opportunities** — code embodying an invariant a spec could usefully protect — while leaving the rest alone.

## Soft-wrap prose convention

The `coach` and `builder` agents now write spec prose as one line per paragraph (soft-wrap) rather than hard-wrapping at a fixed column, keeping spec diffs minimal.

## Upgrade

```bash
spekk update
```
