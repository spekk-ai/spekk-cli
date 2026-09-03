---
id: property-tests
created: 2026-09-03T14:00:00Z
priority: 2
---

# Property Tests Skill

Decides whether a promise in the specs deserves a property-based test, then writes that test for the right layer and proves it reached the state it guards. A property states a promise the product makes and lets a tool search for a state that breaks it: a browser explorer such as Bombadil for the web client, a property-based library such as Hypothesis (Python), Hegel (Rust, Go), or fast-check (TypeScript) for the backend.

**CLI Command:** `spekk coach property-tests [assertion-id | spec-id | layer]`

The method is adapted from the public Antithesis agent skills (antithesishq/antithesis-skills, Apache 2.0), cut down to what transfers to a project that runs its own tools without the Antithesis platform.

## Triggers

- "property test"
- "property-based test"
- "add a property"
- "cover this assertion with a property"
- "fuzz this"
- "Hypothesis test"
- "Bombadil property"
- "false positive in the sweep"
- "triage this property run"
- "which assertions have no test"

## Workflow

### 0. Orient

1. Read the project's runbook for each testing layer it has. A project usually keeps one for its browser sweep and one for its backend property tests; if neither exists, say so and write the property alongside a short runbook rather than in a vacuum.
2. Run `spekk status` to see which specs are complete. Find the assertions that a property could restate with `spekk query "SELECT id, status, title FROM assertions WHERE title LIKE '%keyword%'"`. Only an assertion whose `status` is `done` can back a bug report; a property over a draft assertion is a reach claim at most.
3. Read the project's bug history: closed issues, `observations/`, and any walkthrough or QA reports. A fixed bug is a regression target. A component with no bug history is suspicious, not clean.

### 1. Apply the value gate, and stop if it fails

Search budget, review time, and the reader's attention are finite, and a property that fires on a low-value target costs all three. Answer these in the catalog entry before writing code:

1. **Is the promise `done`?** A property restates a `done` assertion, or it files no issue.
2. **Does it need search?** If a fixed-input unit test settles the invariant, write that test instead. A property earns its place when the input space, the operation order, or the interleaving of states is what hides the bug.
3. **Would the failure matter?** Rank by consequence: data exposed or persisted where it should not be, a record that must be immutable and is not, a permission that leaks, money miscounted, data lost, a server error. A cosmetic difference is not a target.
4. **Is there evidence?** A closed bug or a report ranks above a guess.
5. **Is the cost proportionate?** Score value 1 to 5 as likelihood of a real bug times severity, and cost 1 to 5 as effort to express and run. Write value 4 or 5 first. A value 2 property is worth writing only when its cost is 1 and it runs in seconds.
6. **Is the portfolio balanced?** Three properties on one screen and none on the riskiest path is over-investment. Map each high-risk area to its properties before adding to a well-covered one.

Two anti-patterns deserve their own line. Exhaustive enumeration of a trivial finite space is a slow table test, not property testing; enumerate only when the space is small, the run is fast, and the bug class is real. A property that duplicates a fixture test consumes budget and adds no coverage; grep for the assertion id or the model first, and extend the fixture test if that is all the promise needs.

### 2. Study the code through fixed lenses

Look at the same screens and models through each lens as a fresh pass, then merge. Skipping a lens because an earlier pass "covered that area" is the mistake the method exists to prevent.

| Lens | Backend | Browser |
|---|---|---|
| Data integrity | An aggregate against a naive recount; a constraint a serializer or a bulk path can bypass | A count, a badge, a column, a total, checked against the record behind it |
| Permission boundaries | The role-to-action map against every role subset; the read floor for an account with no role | A control rendered for a role that may not use it; the refusal text as the signature |
| Lifecycle and transitions | Every `(from, to)` pair against the transition map, on every save path | Status badges and the buttons offered, against the map |
| Protocol contracts | List equals detail; a filter's promise equals its rows; `count` equals the rows across pages | A tab equals its definition; list versus detail |
| Idempotency and replay | The same write twice; a create after a soft-delete and restore | A double click, a repeated submit, back and forward |
| Failure paths | A database error that surfaces as a 500 instead of a 400 | A request that fails while the screen shows stale rows and no error |
| Resource boundaries | Query counts on list endpoints | DOM nodes and listeners over a long run |
| Unproven assumptions | `catch` blocks that swallow; "this cannot happen" comments | A placeholder that renders as `undefined` or `Invalid Date` |
| Wildcard | One unconstrained pass at the end | One unconstrained pass at the end |

### 3. Write the catalog entry before the code

One paragraph per property, kept where the project keeps its testing notes: a name, the invariant in one plain sentence, the assertion and its status, how to observe it, the randomness angle (which random inputs or actions reach the state), the reach claim (what proves a run got there), and the value and cost scores. If you cannot write the reach claim, the property is not ready.

### 4. Choose the form

- A safety invariant is checked on every state: `always(...)` in a browser explorer; a plain assertion or an `@invariant()` in a stateful backend test.
- A guarantee after a trigger has a time bound: `always(now(precondition).implies(eventually(...).within(n, "seconds")))` in the browser; a poll with a bound on the backend.
- A reach claim proves the run got somewhere: an exported `eventually(...)` in the browser; an `event("reached: ...")` with the statistics read after the run on the backend.
- Assert bounds, not exact values, where a transient error can move the number, and assert each bound separately so the log says which one broke.
- Never place `eventually(X)` next to `always(!X)`. Assert the precondition, not the violation.

### 5. Implement in the project's house pattern

Follow the existing property file for the layer: one extractor or strategy per fact, constants with names, a comment or docstring that names the assertion. Use only the API the installed tool version ships; a tool's online manual often documents an unreleased branch, and an import the release lacks can take down the whole run at load time.

### 6. Run it clean

Run the new property against seeded data for long enough to reach its state. It must fire zero times unless it found a bug. On a finite input space, run twice: a property that passes once and fails once must enumerate instead of sample. When it fires, decide first whether the property is wrong: read the state or the shrunk example, reproduce it, and fix the property before blaming the product.

### 7. Prove the reach

A quiet run proves nothing until you show the run reached the state where the property could fail: grep the trace for the route or the element, read the event statistics, or run as the account whose role makes the property meaningful.

### 8. Triage and file

A surviving violation gets three tests: the property is one that ships, it reproduces, and no open issue already names it. File one issue with the evidence the project's runbook asks for, name the assertions touched, and mark the test as a strict expected failure that names the issue, so the suite stays green and turns red when the fix lands. Then classify the finding: a defect, a flawed property, or a workload gap where the run never reaches the state.

### 9. Deliver

Small pull requests of three to five properties, with the catalog rows and the issue links. Property tests run after a merge, on a schedule, and on demand, never in the per-push suite, unless the project's runbook says otherwise; state in the pull request how long the run took and what it reached.

## Validation

### Success Criteria

- Every property written names a `done` assertion, or is labeled a reach claim
- Every catalog entry answers the six gate questions and carries value and cost scores
- No property duplicates an existing fixture test for the same promise
- Every property ran clean against seeded data before delivery, and the pull request states the run length and what the run reached
- Every surviving violation has one issue, and the test that found it is a strict expected failure that names the issue

### Example Output

```
Property catalog entry

Name: Next visit implies an open case
Invariant: In every patient row, a non-empty Next Visit cell means Open Cases is at least 1.
Assertion: patient-list-enriched-columns (done)
Observe: the sixth and seventh cells of each row with a data-testid of patient-row-*
Randomness angle: the random clicker reaches the list from three tabs and both sort orders
Reach claim: at least one state with a populated Next Visit cell
Value 4 (this column lied for two months) / Cost 1
Form: always(...) over an extractor that returns the rows as JSON
```

## Notes

- Property tests are one of three test kinds, and the other two stay: scripted end-to-end specs prove a known path still works, and unit tests prove a fixed input gives a fixed answer. Property tests find what nobody wrote a test for.
- The tool split is by question, not by layer: an explorer for the surface a person sees, a property library for the rules underneath.
