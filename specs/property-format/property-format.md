---
id: property-format
created: 2026-11-02T21:30:00Z
priority: 1
---

# Property Format - a v1 Assertion Is a v2 Property

## Problem

In v2 the unit of a spec is a **property**: a rule that must stay true, such as "a user's balance is never below zero". An invariant is one kind of property. A property holds more than a title and success criteria: a kind, the terms it uses, the decisions that tell how this system reads the sentence, the rationale, and, for a property with no checks, the files that hold its code. The parser and the index must read these parts, so that agents can query them and later specs can use them.

A v1 assertion file is a v2 property file. The v1 format stays valid with no change, as a TypeScript file can be a JavaScript file.

## Property format

```markdown
---
id: balance-nonneg
parent: ledger
created: 2026-11-01T00:00:00Z
priority: 1
status: not_started
kind: invariant          # invariant | ordering | response | authorization | example
anchors: [server/ledger/models.py]   # only for properties with no checks
---

# A user's balance is never below zero

## Terms
- balance: `Account.balance`, the ledger balance. Holds are excluded.

## Decisions
- True at every commit. It can go below zero inside a transaction.

## Rationale
We do not offer credit. A negative balance is money that we lent with no agreement.
```

The statement is the title. Terms, decisions and rationale are optional level-2 sections of the body.

## Decisions

- "Property" is the unit, and "invariant" is one kind of property (William, 2026-11-01).
- The kinds come from the property pattern catalog of Dwyer, Avrunin and Corbett (1999). Each kind has a known check shape.
- There is no property DSL in v2.0. The builder writes checks in the test framework of each language.
- A property has two values. The workflow value is the v1 `status`. The truth at HEAD is calculated by spekk (see `verify-command`), and it is never written in the file.
- Front-matter anchors are for properties with no checks.

## Open choices

- **Workflow values.** The brief names them todo, doing and done. v2 keeps the v1 values: `not_started` is todo, `in_progress` is doing, and `done` is done. There is no new enum, so every v1 file stays valid.
- **Required fields.** The draft format in the brief shows no `created`, `priority` or `status`. v2 keeps the v1 required fields, because a change makes v1 tools reject v2 files.
- **No kind.** A property with no `kind` is of kind `example`. This is how a v1 assertion reads.
- **Unknown kind.** An unknown `kind` value gives a warning, not a failure, and the property reads as `example`. v1 projects can already use `kind` as a custom field, and a v1 spec must stay valid.
- **Custom-field rows.** `kind` and `anchors` stay out of the known-key set of the `frontmatter_fields` table, so a v1 query on them keeps working.
- **Anchor paths.** A front-matter anchor is a path to one file, relative to the project root (the directory that holds `specs/`). A directory is not an anchor.

## Assertions

See `assertions/` for what must be true.
