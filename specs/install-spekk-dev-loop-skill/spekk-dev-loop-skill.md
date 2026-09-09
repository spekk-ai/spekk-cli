---
name: spekk-dev-loop
description: Build a real feature on a spekk project in ONE session that plays the coach, builder, and verifier roles in turn — scope into specs, implement the assertions, verify — instead of dispatching a separate subagent per role. Use when asked to scope-and-build a feature, "spec this out and build it", or work a spekk project's assertion queue end to end.
---

# Spekk Dev Loop

Build the feature in **one continuous session** that plays the roles in turn.
Do not dispatch a fresh subagent per role. A pilot showed that, for a feature
that fits in one session, one session uses about 5× fewer tokens, costs about
3× less, and finishes about 3× faster than the coach → builders → review
subagent pipeline, at equal quality (issue #154). The reason is simple: the
build and verify phases reuse the context you built while scoping, instead of a
new agent re-deriving the code each time.

Fetch each role's instructions with `spekk prompt <role>` at the start of its
phase, and follow them in this session. `spekk prompt` already applies the
per-role overrides, so you keep the role's full instructions without a handoff.

Run `spekk init` first if the project has no `specs/` yet.

## Phase 1 — Scope (coach)

Run `spekk prompt coach` and adopt it. Turn the idea into real
`specs/<name>/assertions/*.md`.
- State the problem in your own words, not pre-decided implementation details.
- **Lean and simple, every time**: the simplest thing that solves the real
  problem, not general or configurable machinery. Name the concrete temptations
  you reject.
- Ground in the real current source, never a paraphrase.
- Mark an assertion `not_started` when it is concretely buildable; mark it
  `draft` only for a real, named design gap — not as a cautious default.
- Confirm the specs parse: `spekk validate` (or `spekk status`/`next`).

## Phase 2 — Build

Run `spekk prompt builder` and adopt it. Implement the assertions in dependency
order (`spekk next`), in this same session.
- You already understand the code from Phase 1. Reuse that context; do not
  re-derive it.
- Implement one assertion at a time. Read each assertion file as the
  authoritative spec.
- Keep the lean-and-simple mandate.
- Verify each assertion against its real success criteria — a live check against
  a running service when the assertion calls for one, not just "it compiles".
- Mark each assertion `done` when it is implemented and tested.

## Phase 3 — Verify

Run `spekk skill show builder review` and follow it in this session. It reviews the assertions this session marked `done` against the real code, applies six lenses with a remedy each, and fixes what it finds. Tests must be lean and high value: delete a low-value or redundant test rather than keep it.

A self-review reuses the context that the build made. That is what makes it cheap, and it is also what it loses: independence. For a high-stakes or large change, run `spekk builder review` instead, so a fresh session reads the code with no memory of writing it. That is the exception, not the default.

Loop Phases 2–3 until the queue is empty. Commit as you go. Push only after the review reports. That is also where you catch a judgment call you disagree with, before it is live.

## Escalation — when one session is not enough

The single session is the default for a feature that fits in one session. It is
not the only path. Escalate to sub-agents when:
- the feature is too large for one coherent session (context strain or loss of
  coherence), or
- the user asks for a parallel workflow.

In that case, dispatch builders in separate worktrees on genuinely independent
paths, and integrate and verify their work yourself. Treat this as the
exception, not the default — for most features, one session wins.
