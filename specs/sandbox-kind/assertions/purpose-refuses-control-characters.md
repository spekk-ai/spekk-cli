---
id: purpose-refuses-control-characters
parent: sandbox-kind
created: 2026-10-16T01:10:00Z
priority: 1
status: done
depends-on: one-off-create-labels-and-marker
branch: sandbox-kind
---

# `--purpose` Is One Line of Text

`status` prints the purpose as a `Key: value` line. A newline in the purpose breaks that layout, and an escape sequence goes to the terminal of each person who runs `status`. The purpose is a short reason, so a control character in it is always an error.

**Tests:** internal/sandbox/kind_test.go

## Success Criteria

- `spekk sandbox create --purpose <text>` refuses a purpose that has a control character: a C0 character (which includes newline, tab and ESC), DEL, or a C1 character. The create stops before anything bills, with one error line that names the character.
- All other characters are accepted. Quotes, `$()`, backticks, `;` and non-ASCII letters arrive byte for byte in `sandboxes.json` and in `/opt/spekk/sandbox.json`.

## Verification

- A test checks a newline, a tab, ESC, DEL and a C1 character (refused), and a purpose with quotes, `$()`, backticks, `;` and a non-ASCII letter (accepted).
- A walkthrough on a machine you already have: a purpose with a newline is refused, and the hostile purpose without control characters arrives byte for byte in the record and in the marker, and nothing in it runs on the machine.
