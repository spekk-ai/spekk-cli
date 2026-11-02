---
id: builder-prompt-builds-evidence
parent: property-prompts
created: 2026-11-02T21:30:00Z
priority: 2
status: not_started
---

# The Builder Prompt Teaches Evidence Markers, Anchors and `spekk verify`

## Description

The builder makes a property true and gives the evidence. Its prompt must make each check name its property, put anchors only at enforcement points, and keep the properties of a file true when it edits the file.

## Success criteria

- The builder prompt calls the unit a "property" and does not call specs "declarative".
- It tells the builder to write each check in the test framework of the language, with an evidence marker that names the property and its level, and shows the Go and the pytest marker of `evidence-markers`.
- It tells the builder to put a comment anchor only at an enforcement point (the guard, the database constraint, the lock), and not on each line that touches the concept.
- It tells the builder to run `spekk constraints <file>` before it edits a file, and to keep each listed property true.
- It tells the builder to run `spekk verify` before it marks a property `done` when `spekk.toml` has a `[[verify.run]]`, and to fix a `failing` property before it commits.
- It says that `done` means "built", and that only the truth at HEAD says "checked". The builder never writes a truth or a level in a property file.
- `spekk prompt builder` prints the new text.
