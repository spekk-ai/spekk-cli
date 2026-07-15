package main

import "fmt"

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
