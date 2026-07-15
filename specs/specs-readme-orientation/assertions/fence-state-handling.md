---
id: fence-state-handling
parent: specs-readme-orientation
created: 2026-07-15T00:00:00Z
priority: 1
status: done
branch: feat/specs-readme-orientation
depends-on: idempotent-regeneration
---

# Regeneration Upgrades Legacy READMEs and Recovers From a Corrupt Fence

An existing `specs/README.md` may have no managed fence at all (a legacy static
README, or a hand-written one) or a damaged fence (a marker deleted, duplicated,
or reordered). Regeneration must handle every case without destroying human
prose, converging on exactly one well-formed managed region.

## Success Criteria

- **No fence present (legacy/static upgrade).** When the README exists but
  contains neither marker, regeneration preserves the entire existing file and
  appends a fresh managed region as the final block, separated from the prior
  content by exactly one blank line, ending in a single trailing newline. The
  pre-existing text (including the current static `specsReadme` content, if that
  is what is on disk) is left byte-for-byte intact.
- **Well-formed fence present.** Exactly one begin marker followed by exactly one
  end marker is the happy path handled by `idempotent-regeneration` (replace
  in place). Regeneration must not create a second region in this case.
- **Corrupt fence (regenerate the block, do not patch).** A "corrupt" fence is
  any state that is not the well-formed one: a begin marker with no end marker
  after it, an end marker with no preceding begin marker, duplicate begin or end
  markers, or an end marker appearing before a begin marker. In every corrupt
  case the CLI removes the stray marker line(s) and any bytes the markers
  enclose (that span is CLI-owned and disposable), then appends one fresh,
  well-formed managed region. It never attempts a partial patch.
- **Convergence.** After one regeneration pass over any of the above states, the
  file contains exactly one begin marker and exactly one end marker in order,
  and a second pass is byte-identical (ties back to `idempotent-regeneration`).

**Note:** The corrupt-fence rule deliberately favors safety of *recognizable*
human prose (text clearly outside any marker) over trying to rescue text
sandwiched inside a broken fence — that inner text is treated as CLI-owned. This
is the "regenerate the whole block rather than patch" contract from the issue.
Keep detection to literal marker-line counting/ordering; no fuzzy parsing.

## Tests

Table-driven Go test with one case per state: (a) legacy static README with no
markers → original text preserved + one region appended; (b) only a begin
marker; (c) only an end marker; (d) duplicate begin markers; (e) end-before-begin.
Each asserts the post-pass file has exactly one well-formed region, that text
outside any marker is preserved, and that a second pass is byte-identical.

**Tests:** cmd/spekk/readme_test.go (TestRegenerateReadmeContent_FenceStates,
TestRegenerateReadmeContent_LegacyUpgradeAppendsOneRegion),
cmd/spekk/init_test.go (TestRunInit_UpgradesLegacyReadme,
TestRunInit_RecoversCorruptFence)
