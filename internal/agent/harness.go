package agent

import (
	"fmt"
	"sort"
	"strings"
)

// Profile describes how to launch an agent under a particular harness (the CLI
// tool that actually runs the coach, builder, or observer). Every value that a
// launch site used to embed as a `"claude"` literal — the binary name, the
// flags, and the not-found guidance — lives here instead, so selecting a
// different harness is a matter of resolving a different profile.
type Profile struct {
	// Name is the canonical harness name (e.g. "claude-code").
	Name string
	// Binary is the executable spawned at every launch site.
	Binary string
	// DisplayName is how the harness is named in user-facing messages
	// (e.g. "Claude Code").
	DisplayName string
	// InstallURL points at the harness's install instructions, shown when the
	// binary is missing.
	InstallURL string
	// PromptFlags precede the prompt argument. Nil means the prompt is passed as
	// a bare trailing positional argument.
	PromptFlags []string
	// SkipPermissionsFlags skip the harness's interactive permission prompts.
	SkipPermissionsFlags []string
	// HeadlessFlags request non-interactive/print mode (no TTY).
	HeadlessFlags []string
	// SystemPromptFlags precede the prompt when it is passed as a system prompt
	// rather than an initial user message (the interactive builder).
	SystemPromptFlags []string
}

// claudeCodeProfile is the built-in default: the flags and guidance that were
// hardcoded across the launch sites before harnesses became selectable.
var claudeCodeProfile = Profile{
	Name:                 "claude-code",
	Binary:               "claude",
	DisplayName:          "Claude Code",
	InstallURL:           "https://claude.ai/code",
	PromptFlags:          nil,
	SkipPermissionsFlags: []string{"--dangerously-skip-permissions"},
	HeadlessFlags:        []string{"-p"},
	SystemPromptFlags:    []string{"--system-prompt"},
}

const defaultHarness = "claude-code"

// harnessProfiles maps canonical harness names to their profiles.
var harnessProfiles = map[string]Profile{
	claudeCodeProfile.Name: claudeCodeProfile,
}

// harnessAliases maps alternative names to their canonical harness name.
var harnessAliases = map[string]string{
	"claude": "claude-code",
}

// DefaultProfile returns the built-in default harness profile (claude-code).
func DefaultProfile() Profile {
	return claudeCodeProfile
}

// ResolveProfile returns the profile for a harness name. An empty name resolves
// to the default; aliases resolve to their canonical profile. An unknown name
// returns an error listing the valid names rather than a zero profile a caller
// might spawn.
func ResolveProfile(name string) (Profile, error) {
	if name == "" {
		name = defaultHarness
	}
	if canonical, ok := harnessAliases[name]; ok {
		name = canonical
	}
	p, ok := harnessProfiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("unknown harness %q; valid harnesses are: %s", name, strings.Join(knownHarnessNames(), ", "))
	}
	return p, nil
}

// knownHarnessNames returns the canonical harness names plus aliases, sorted,
// for use in error messages.
func knownHarnessNames() []string {
	names := make([]string, 0, len(harnessProfiles)+len(harnessAliases))
	for name := range harnessProfiles {
		names = append(names, name)
	}
	for alias := range harnessAliases {
		names = append(names, alias)
	}
	sort.Strings(names)
	return names
}

// promptArgs returns the prompt argument preceded by any PromptFlags.
func (p Profile) promptArgs(prompt string) []string {
	return append(append([]string{}, p.PromptFlags...), prompt)
}

// InteractiveArgs returns the argv (excluding the binary) to launch the harness
// interactively, skipping permissions, with prompt as the initial message.
func (p Profile) InteractiveArgs(prompt string) []string {
	args := append([]string{}, p.SkipPermissionsFlags...)
	return append(args, p.promptArgs(prompt)...)
}

// SystemPromptArgs is like InteractiveArgs but passes the prompt as a system
// prompt (used by the interactive builder so the harness waits for user input).
func (p Profile) SystemPromptArgs(prompt string) []string {
	args := append([]string{}, p.SkipPermissionsFlags...)
	args = append(args, p.SystemPromptFlags...)
	return append(args, prompt)
}

// HeadlessArgs returns the argv (excluding the binary) to run the harness in
// non-interactive/headless mode with prompt.
func (p Profile) HeadlessArgs(prompt string) []string {
	args := append([]string{}, p.HeadlessFlags...)
	args = append(args, p.SkipPermissionsFlags...)
	return append(args, p.promptArgs(prompt)...)
}

// notFoundLines returns the two guidance lines shown when the harness binary is
// missing: the "not found — install X" line and the install URL line.
func (p Profile) notFoundLines() (string, string) {
	return fmt.Sprintf("%s CLI not found. Please install %s first.", p.DisplayName, p.DisplayName),
		fmt.Sprintf("Visit: %s for installation instructions.", p.InstallURL)
}
