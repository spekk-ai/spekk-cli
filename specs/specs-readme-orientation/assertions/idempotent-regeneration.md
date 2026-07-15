---
id: idempotent-regeneration
parent: specs-readme-orientation
created: 2026-07-15T00:00:00Z
priority: 1
status: not_started
branch: feat/specs-readme-orientation
depends-on: managed-readme-block
---

# Regenerating the Managed Block Is Idempotent and Preserves Human Prose

Running `spekk init` against a `specs/README.md` that already contains a valid
managed region rewrites only the bytes between the markers, leaving all human
prose outside the fence untouched. This assertion owns the valid-fence
replace-in-place path and the byte-level idempotency guarantee.

## Success Criteria

- **`spekk init` regenerates on an already-initialized project.** Today `runInit`
  early-returns when `specs/` already exists (it prints "specs/ already exists —
  you're set." and never touches any file), so the regeneration path below is
  unreachable through the CLI. This assertion changes that entry point: when
  `specs/README.md` already exists, `spekk init` proceeds to (re)generate its
  managed region in place rather than no-op'ing. The reassuring "already set up"
  messaging may stay, but the README's managed block is now (re)rendered on every
  run, so schema drift shows up as a diff the next time `init` runs.
- **Replace between markers only.** Given a README with exactly one begin marker
  followed by exactly one end marker, regeneration replaces the span from the
  begin-marker line through the end-marker line (inclusive) with the freshly
  rendered managed region. Bytes before the begin marker and after the end
  marker are preserved verbatim, including any human headings, paragraphs, and
  blank lines.
- **Byte-identical idempotency.** Running the regeneration twice in a row with no
  change to the schema version constant produces a byte-identical
  `specs/README.md` on the second run — no trailing-whitespace drift, no added or
  removed blank lines, no newline-count change at the marker boundaries or EOF.
- **Marker-adjacency is stable.** The blank-line spacing the CLI emits
  immediately inside/around the markers is fixed by the renderer, so repeated
  runs neither accumulate nor strip blank lines adjacent to the fence.
- **Version bump produces a diff, not corruption.** If the schema version
  constant changes, regeneration replaces the managed span with the new content
  and the human prose outside is still preserved verbatim.
- **String-splitting only.** Locating the region uses marker-delimited string
  searching (find begin marker line, find end marker line after it, splice) — no
  markdown/HTML parser and no line-by-line diff/patch.

**Note:** "Idempotent" here is byte-exact, not merely semantically equal. The
test must compare full file bytes, because a stray trailing space or an extra
newline is exactly the kind of drift that makes committed READMEs churn in git.

## Tests

Go test: build a README = `human intro` + valid managed region + `human
outro`; run regeneration; assert (1) the outer text is byte-identical to the
input outside the fence, (2) the managed region equals the current render, and
(3) a second regeneration yields output byte-identical to the first. Add a case
that bumps the version constant and asserts the outer prose still survives.
