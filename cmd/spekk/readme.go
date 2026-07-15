package main

import (
	"fmt"
	"strings"
)

// specSchemaVersion is the current version of the specs/ frontmatter schema
// documented in the managed region of specs/README.md. Bump it whenever the
// schema in internal/parser changes. Because the managed block is a pure
// function of this constant, bumping it is the entire schema-drift
// detection mechanism: the next `spekk init` (or future in-place
// regeneration) renders different bytes, so the change shows up as a plain
// git diff. No separate "compare versions" command is needed.
const specSchemaVersion = 1

// Fence markers delimiting the CLI-owned region of specs/README.md. Other
// assertions (idempotent regeneration, fence-state handling) locate and
// replace the region using these two literal strings.
const (
	readmeManagedBeginMarker = "<!-- spekk:managed:begin -->"
	readmeManagedEndMarker   = "<!-- spekk:managed:end -->"
)

// specsReadmeIntro is the human-facing prose written above the managed
// region on fresh init. Unlike the managed region, it is not CLI-owned —
// readers are free to edit it.
const specsReadmeIntro = `# Specs

This directory is a work queue for AI agents, managed with
[spekk](https://github.com/spekk-ai/spekk-cli).
`

// specsReadmeManagedBody returns the content rendered between
// readmeManagedBeginMarker and readmeManagedEndMarker: the specs/ concept
// model, the full frontmatter schema (matching internal/parser exactly),
// and the current schema version.
//
// It MUST be a pure function of specSchemaVersion — no spec names, counts,
// statuses, timestamps, or other per-project data. Two different projects
// on the same schema version must render byte-identical managed blocks.
func specsReadmeManagedBody() string {
	return fmt.Sprintf(`## Concepts

specs/ is a work queue: a flat list of specs, each describing a feature or
change that must become true. Each spec is a folder, specs/<spec-name>/,
containing <spec-name>.md — a file stating what must be true and why. Every
spec folder also has an assertions/ subfolder holding one file per small,
independently testable step toward that spec.

The command loop:

  spekk init      creates and maintains this directory
  spekk coach     turns ideas and requests into specs and assertions
  spekk builder   implements the next ready assertion (wraps spekk next)
  spekk next      prints the next ready assertion without implementing it
  spekk status    shows an overview of every spec and assertion

## Frontmatter reference

Every spec file (<spec-name>.md) has this frontmatter:

  id          required, kebab-case (e.g. my-feature)
  created     required, ISO-8601 timestamp: YYYY-MM-DDTHH:MM:SSZ
  priority    required, integer 1-3 (1 is highest)
  status      optional, defaults to not_started
  branch      optional, defaults to main

Every assertion file (assertions/<assertion-name>.md) has the same fields,
plus:

  parent      required, id of the spec this assertion belongs to
  depends-on  optional, id of another assertion that must be done first
  locked-by   optional, set while a builder has the assertion claimed as
              in_progress; removed once it leaves that status

The valid status values are: not_started, in_progress, done, draft, failed.

## Schema version

spekk_schema_version: %d

This number identifies the frontmatter schema documented above. It changes
only when that schema changes, so diffing specs/README.md across a spekk
upgrade is enough to detect drift.
`, specSchemaVersion)
}

// renderManagedReadmeBlock returns the fenced, CLI-owned region of
// specs/README.md: the begin marker, the managed body, and the end marker.
func renderManagedReadmeBlock() string {
	return readmeManagedBeginMarker + "\n" + specsReadmeManagedBody() + readmeManagedEndMarker + "\n"
}

// renderSpecsReadme returns the full contents of specs/README.md for a
// fresh spekk init: human intro prose followed by the managed region. The
// result always ends in exactly one trailing newline.
func renderSpecsReadme() string {
	return specsReadmeIntro + "\n" + renderManagedReadmeBlock()
}

// spliceManagedReadmeBlock replaces the span from the begin-marker line
// through the end-marker line (inclusive) in content with newBlock, using
// marker-delimited string searching only (no markdown/HTML parser, no
// line-by-line diff). Everything before the begin marker and everything
// after the end marker's line is preserved byte-for-byte.
//
// ok is false when content does not contain a well-formed fence — a begin
// marker followed later by an end marker — in which case content is
// returned unchanged. Callers that need to handle malformed or missing
// fences do so themselves; this function only knows the happy path.
func spliceManagedReadmeBlock(content, newBlock string) (result string, ok bool) {
	beginIdx := strings.Index(content, readmeManagedBeginMarker)
	if beginIdx == -1 {
		return content, false
	}
	endIdx := strings.Index(content[beginIdx:], readmeManagedEndMarker)
	if endIdx == -1 {
		return content, false
	}
	endIdx += beginIdx

	// newBlock already supplies the newline that terminates the end
	// marker's line, so drop the corresponding newline from what follows
	// in content to avoid doubling it.
	after := strings.TrimPrefix(content[endIdx+len(readmeManagedEndMarker):], "\n")

	return content[:beginIdx] + newBlock + after, true
}

// replaceManagedReadmeBlock regenerates the managed region of an existing
// specs/README.md in place, using the current renderManagedReadmeBlock
// output. See spliceManagedReadmeBlock for the splice contract and the
// meaning of ok.
func replaceManagedReadmeBlock(content string) (result string, ok bool) {
	return spliceManagedReadmeBlock(content, renderManagedReadmeBlock())
}

// appendManagedReadmeBlock appends a fresh managed region to existing, as
// the final block, separated from it by exactly one blank line and ending
// in a single trailing newline. If existing is empty (or all newlines), the
// managed region is the entire result — no leading blank line to nothing.
//
// Used both to upgrade a legacy/static README that has no fence yet, and to
// reassemble a README after a corrupt fence has been stripped down to the
// human prose around it.
func appendManagedReadmeBlock(existing string) string {
	trimmed := strings.TrimRight(existing, "\n")
	if trimmed == "" {
		return renderManagedReadmeBlock()
	}
	return trimmed + "\n\n" + renderManagedReadmeBlock()
}

// regenerateReadmeContent is the single entry point spekk init uses to
// bring an existing specs/README.md in line with the current managed
// block, no matter what fence state it's in:
//
//   - Well-formed fence: exactly one begin marker followed later by exactly
//     one end marker. Delegates to replaceManagedReadmeBlock, the
//     idempotent-regeneration happy path (replace in place).
//   - No fence at all: a legacy/static README (or a hand-written one with
//     no markers). The entire file is preserved and a fresh managed region
//     is appended as the final block.
//   - Anything else is corrupt: a begin with no end after it, an end with
//     no begin before it, duplicate begins or ends, or an end before a
//     begin. The CLI never patches a corrupt fence — it deletes every
//     marker line plus everything the markers (would have) enclosed, then
//     appends one fresh, well-formed managed region after whatever prose
//     remains outside that span. When one marker type is entirely absent
//     (a dangling begin or a dangling end), the enclosed span extends all
//     the way to the natural boundary it's missing — EOF for a begin with
//     no end, the start of the file for an end with no begin — since that
//     span is exactly what a previous, now-broken managed region would
//     have occupied. Otherwise (both types present at least once) the
//     span runs from the earliest to the latest marker line.
//
// Detection and stripping use marker-line identity and index arithmetic
// only — no markdown/HTML parsing, no fuzzy matching. changed reports
// whether result differs from content.
func regenerateReadmeContent(content string) (result string, changed bool) {
	lines := strings.Split(content, "\n")

	var beginLines, endLines []int
	for i, line := range lines {
		switch line {
		case readmeManagedBeginMarker:
			beginLines = append(beginLines, i)
		case readmeManagedEndMarker:
			endLines = append(endLines, i)
		}
	}

	wellFormed := len(beginLines) == 1 && len(endLines) == 1 && beginLines[0] < endLines[0]
	noFence := len(beginLines) == 0 && len(endLines) == 0

	switch {
	case noFence:
		updated := appendManagedReadmeBlock(content)
		return updated, updated != content

	case wellFormed:
		updated, _ := replaceManagedReadmeBlock(content)
		return updated, updated != content

	default:
		var first, last int
		switch {
		case len(endLines) == 0:
			// Dangling begin(s), no end anywhere: the span they would
			// have enclosed runs to EOF.
			first, last = beginLines[0], len(lines)-1
		case len(beginLines) == 0:
			// Dangling end(s), no begin anywhere: the span they would
			// have enclosed runs back to the start of the file.
			first, last = 0, endLines[len(endLines)-1]
		default:
			// Both marker types appear at least once (duplicates and/or
			// out of order): strip from the earliest to the latest
			// marker line.
			markerLines := append(append([]int{}, beginLines...), endLines...)
			first, last = markerLines[0], markerLines[0]
			for _, m := range markerLines[1:] {
				first = min(first, m)
				last = max(last, m)
			}
		}

		preserved := append(append([]string{}, lines[:first]...), lines[last+1:]...)
		outer := strings.Join(preserved, "\n")

		updated := appendManagedReadmeBlock(outer)
		return updated, true
	}
}
