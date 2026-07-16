---
name: spekk-dev-loop
description: William's outer dev loop for building real features on a spekk-enabled project via a coach → coordinate → builders → review pipeline of subagents. Use when asked to scope-and-build a feature, "spec this out and build it", run the coach-then-builders flow, or work through a spekk project's assertion queue end to end.
---

# Spekk Dev Loop

Four subagent stages, each a fresh dispatch (not one continued conversation),
for turning an idea into shipped, verified code on a spekk project (`specs/`,
`spekk status`/`next`). Run `spekk init` first if the project has no
`specs/` yet, or `spekk install --target <harness>` if the coach/builder
roles aren't registered for the current assistant.

## 1. Coach (opus) — scope into specs

Turn the idea into real `specs/<name>/assertions/*.md`. Always opus — bad
scoping decisions are cheapest to catch here. Give it:
- The problem in your own words, not pre-decided implementation details.
- **The lean-and-simple mandate, every time**: simplest thing that solves the
  real problem, not general/configurable machinery. Name concrete
  temptations to reject.
- Pointers to the real current source, never a paraphrase.
- What's genuinely worth asking you vs. what it should just decide and defend.

Mark assertions `ready`/`not_started` when concretely buildable; `draft` only
for a real, named design gap — not as a cautious default.

## 2. Coordinate (coach skill, also opus) — review, revise, sequence

A second, independent opus pass over stage 1's specs — not a rubber stamp.
Check dependency order, catch correctness issues (races, wrong assumptions,
assertions blocked on requirements that don't exist), fix underspecified
assertions directly. Finish with `spekk status`/`next` to confirm it parses
and get the ordered queue. Commit, don't push yet.

## 3. Builders (sonnet) — implement one assertion at a time

One builder per assertion, sequential — never start the next until the
current one is implemented, tested, and committed, even if `spekk next`
shows several ready. Each builder grounds itself in the *current* code, not
a summary of prior builders' work. Give it:
- The assertion file path as the authoritative spec — read it directly.
- Real current source pointers, and the lean-and-simple mandate.
- Instruction to verify against the assertion's actual success criteria,
  including a live check against a running service if that's what it calls
  for — not just "it compiles."
- Commit when done, **don't push** — the push gate is yours.

Work touching something outside the repo (sibling repo, live service, DB):
say so explicitly, and require the same "verify for real, clean up after
yourself" bar — disposable test resources over production credentials, with
cleanup confirmed.

## 4. Review (opus)

After the queue empties (or per spec group on a large batch), one opus pass
over what actually got built, running result where possible. Apply the
standing test bar every time, unprompted:

> **Tests must be LEAN and HIGH VALUE only.** Delete low-value or redundant
> tests rather than preserving them — see `code-quality-principles`'s "Lean,
> high-value tests only" for the full rule.

Flag/fix: tests restating the implementation instead of pinning down real
behavior, duplicate coverage, and tests that don't exercise the real code
path.

## Verification discipline (every stage, not just review)

Never push a subagent's work on its self-report alone.
- After coach/coordinate: confirm specs actually parse (`spekk status`/`next`).
- After each builder: independently rebuild, vet, and test yourself. If the
  assertion calls for a live check, reproduce the key confirmation yourself
  before trusting it.
- Push only after your own verification — this is also where you catch a
  subagent's judgment calls you disagree with, before they're live.

Run stages sequentially by default. Reach for true parallel multi-agent
orchestration only when the user has explicitly opted into a workflow for
this task.
