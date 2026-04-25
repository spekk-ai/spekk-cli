// Package spekk provides embedded assets for the spekk CLI binary.
package spekk

import "embed"

// PromptFS contains the built-in agent prompt files, embedded at compile time.
// This allows the binary to resolve default prompts without needing the source
// tree on disk (e.g., when installed via `go install`).
//
//go:embed specs/coach-agent/coach.prompt.md specs/builder-agent/builder.prompt.md specs/observer-agent/observer.prompt.md
var PromptFS embed.FS
