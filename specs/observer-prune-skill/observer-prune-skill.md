---
id: observer-prune-skill
created: 2026-07-25T12:00:00Z
priority: 2
---

# Observer Prune Skill (Anti-Sprawl)

The spekk dev loop is **additive by construction** — coach adds assertions, builders add code, the default observer checks that code still matches specs. Nothing in the loop is charged with asking *"what should be REMOVED?"* So codebases accumulate dead code, duplication, speculative abstractions, and unused flags with no counter-pressure.

This spec adds an **opt-in observer skill** named `prune` that surfaces genuinely-unused code and design-level redundancy as candidates for human review. It is a **skill**, not core drift, because deletion and consolidation are judgment-heavy **architecture/design decisions** (precision-critical — a false "delete this" is dangerous), best applied deliberately rather than on every scan.

**Crucially, `prune` reasons about usage and design — not spec coverage.** spekk specs are *progressive*: adoption is encouraged but always incomplete, so the vast majority of code legitimately has no owning assertion. The **absence of a spec is therefore never a signal to delete**. This is what distinguishes `prune` from `coverage-gap`: `coverage-gap` encourages *documenting* used code that lacks a spec (advancing adoption), whereas `prune` flags code that is genuinely *unused* (no caller/test/reference) or redundant *by design*. The two overlap only at truly dead code — worth neither documenting nor keeping — and `prune` never treats "no spec" as evidence of that.

## Architecture (matches the existing observer-skill pattern)

- A package observer skill is **one markdown file** in `specs/observer-skills/` (template: `coverage-gap-skill.md`). Prompt/skill-driven, launched via the `claude` CLI. **No Go drift engine, no AST walker** — the code-side analysis is done by the LLM, exactly as `coverage-gap` does it.
- Ships by adding the file's path to the `//go:embed` directive at `embedded.go:11` (explicit-path embed, not a glob).
- Discovery is automatic once file + embed exist (`internal/cli/skill.go` `ResolveSkill` / `ListSkills`); invocable as `spekk observer prune`; help lists it dynamically.
- Observations use the shared **Observation Output Contract** (`specs/observer-agent/observer.prompt.md` and `specs/observer-skill-discovery/observer-skill-discovery.md`). The skill writes to `observations/prune/YYYY-MM-DDTHH-MM-SSZ.md`; `consolidate` / `DIGEST.md` enumerate `observations/*/` automatically (no change needed).
- The observer is **HARD READ-ONLY** except `observations/`. `prune` can only RECOMMEND — it must never delete code.

## Naming & type decisions (decided, not open)

- **`prune`, not `simplify`.** The skill's angle is removing genuinely-unused code and cutting design-level redundancy. `prune` names that sharply. `simplify` is broader, overlaps with general refactoring, and blurs the "what is actually unused / redundant" focus that makes this skill useful and safe.
- **One observation type `prune_candidate`, not a deletion/consolidation split.** The `type` field is coarse routing for `consolidate`/`DIGEST`; sub-classification (deletion vs duplication vs over-abstraction vs dead flag) lives in the observation body and severity, mirroring how `coverage-gap` uses a single `coverage_gap` type across many finding kinds. One type keeps the allowed-type lists and downstream consolidation lean.

## Scope (lean)

Four assertions: the skill markdown, the registered observation type, embedding into the binary, and discovery (resolves + lists) verified by a Go test that mirrors the existing `observer-skill-discovery` tests. No new top-level CLI command (free via discovery), no new flag, no Go drift engine. The read-only contract is not weakened; `coverage-gap` and `consolidate` are not touched.
