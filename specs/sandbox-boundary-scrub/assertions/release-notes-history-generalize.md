---
id: release-notes-history-generalize
parent: sandbox-boundary-scrub
created: 2026-07-23T00:00:00Z
priority: 3
status: done
---

# Published Release Notes Generalize Control-Host Internals

**Decision (defended below):** soften the wording in place — do not silently
rewrite substance, and do not leave the leak in place.

`docs/release-notes/RELEASE-NOTES-1.3.0.md` is published history in a public
repo. Its "admin use only" note named the control host's implementation stack
("partially automated through ...") and presented one chat surface as *the*
integration ("Connecting to Slack is a separate process"). Per this spec's
boundary, the server is referred to only generically, and a chat surface is
named only as one example — never as the implementation.

Because these are published notes, the tension is: scrub the leak without
falsifying the record of what shipped. The resolution is a **meaning-preserving
generalization**, not a deletion or a rewrite of substance. The sentence still
says the same thing happened (agent deploy was separate from wiring up the chat
integration, and that wiring was partially automated server-side), only without
asserting the private stack name or fixing a single chat surface as the sole
integration. To keep the edit transparent rather than a silent revision of the
record, a short dated editor's note records that the file was lightly edited
post-publication to generalize control-host internals.

## Success Criteria

- `RELEASE-NOTES-1.3.0.md` no longer names the control host's implementation
  stack: a case-insensitive search for the stack name in the file returns
  nothing.
- It no longer presents a single chat surface as *the* integration. If a chat
  surface is mentioned at all, it appears as one example (e.g. "a chat surface
  such as Slack"), not as the implementation.
- The generalized wording preserves the original factual claim about what
  shipped (agent deploy separate from the server-automated integration step) —
  softened, not contradicted or removed.
- A short dated editor's note in the file records that it was edited
  post-publication to generalize control-host details, so the change to
  published history is transparent, not a silent revision.
- The sweep is re-run across all of `docs/release-notes/` for the stack name
  and for chat surfaces framed as the sole integration. At authoring time only
  `RELEASE-NOTES-1.3.0.md` matched; any sibling that surfaces later gets the
  same treatment.
