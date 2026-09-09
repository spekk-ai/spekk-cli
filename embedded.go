// Package spekk provides embedded assets for the spekk CLI binary.
package spekk

import "embed"

// EmbeddedFS contains the built-in agent prompts and packaged skill files,
// embedded at compile time. This allows the binary to resolve default prompts
// and skills without needing the source tree on disk (e.g., when installed via
// `go install`).
//
//go:embed specs/coach-agent/coach.prompt.md specs/builder-agent/builder.prompt.md specs/observer-agent/observer.prompt.md specs/coach-skills-system/coordinator-skill.md specs/coach-skills-system/meeting-notes-to-specs-skill.md specs/coach-skills-system/business-model-validator-skill.md specs/coach-skills-system/property-tests-skill.md specs/observer-skills/coverage-gap-skill.md specs/observer-skills/consolidate-skill.md specs/observer-skills/prune-skill.md specs/install-spekk-dev-loop-skill/spekk-dev-loop-skill.md specs/builder-skills/review-skill.md
var EmbeddedFS embed.FS
