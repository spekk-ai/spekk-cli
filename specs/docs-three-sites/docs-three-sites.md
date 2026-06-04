---
id: docs-three-sites
created: 2026-06-04T14:00:00Z
priority: 2
---

# Three-Site Documentation

Organize documentation into three Zensical sites by purpose: user docs (how to use spekk), engineering docs (how spekk works internally), and QA docs (release testing and verification).

## Context

- Existing `docs/` site is user-facing and well-structured — it stays as the public/user docs
- Engineering docs cover architecture, internals, and contributing — for people building spekk
- QA docs cover release testing checklists and manual verification — for validating releases before shipping
- All three sites use Zensical and cross-link to each other
- Audiences overlap (all engineers) but purposes are distinct

## Assertions

1. `engineering-docs-site` — `docs-engineering/` Zensical site with architecture, contributing guide, and internals
2. `qa-docs-site` — `docs-qa/` Zensical site with release testing checklist and platform verification
3. `user-docs-gaps` — Existing `docs/` site gets troubleshooting, expanded observer docs, and update command docs
4. `docs-nav-cross-linking` — Each site links to the other two (depends on 1, 2, 3)
