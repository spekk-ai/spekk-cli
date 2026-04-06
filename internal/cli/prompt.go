// Package cli provides CLI utilities for the spekk tool, including prompt
// resolution for coach, builder, and observer agents.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PromptSeparator is the delimiter placed between concatenated prompt layers.
const PromptSeparator = "\n\n---\n\n"

// validAgents lists the recognised agent names.
var validAgents = map[string]bool{
	"coach":    true,
	"builder":  true,
	"observer": true,
}

// PromptResolver resolves layered agent prompts using override and extension
// files from the local project, global home directory, and the spekk
// installation directory.
type PromptResolver struct {
	// InstallDir is the root of the spekk installation (contains specs/).
	InstallDir string
	// WorkDir is the current working directory (project root).
	WorkDir string
	// HomeDir is the user's home directory.
	HomeDir string
}

// ResolvePrompt resolves the full prompt content for the given agent using
// layered resolution:
//
//  1. Base prompt (first match wins):
//     - Local override:  <WorkDir>/.spekk/<agent>.prompt.override.md
//     - Global override: <HomeDir>/.spekk/<agent>.prompt.override.md
//     - Package base:    <InstallDir>/specs/<agent>-agent/<agent>.prompt.md
//
//  2. Extension layers (appended in order):
//     - Global extend: <HomeDir>/.spekk/<agent>.prompt.md
//     - Local extend:  <WorkDir>/.spekk/<agent>.prompt.md
//
// Missing override/extend files are silently skipped. A missing package base
// prompt (when no override is found) is a fatal error.
func (pr *PromptResolver) ResolvePrompt(agent string) (string, error) {
	if !validAgents[agent] {
		return "", fmt.Errorf("unknown agent: %s", agent)
	}

	localDir := filepath.Join(pr.WorkDir, ".spekk")
	globalDir := filepath.Join(pr.HomeDir, ".spekk")

	// Step 1: Determine base prompt (first match wins).
	localOverridePath := filepath.Join(localDir, agent+".prompt.override.md")
	globalOverridePath := filepath.Join(globalDir, agent+".prompt.override.md")
	packageBasePath := filepath.Join(pr.InstallDir, "specs", agent+"-agent", agent+".prompt.md")

	var base string
	if content, err := readIfExists(localOverridePath); err == nil && content != "" {
		base = content
	} else if content, err := readIfExists(globalOverridePath); err == nil && content != "" {
		base = content
	} else {
		content, err := readIfExists(packageBasePath)
		if err != nil {
			return "", fmt.Errorf("prompt file not found: %s", packageBasePath)
		}
		if content == "" {
			return "", fmt.Errorf("prompt file not found: %s", packageBasePath)
		}
		base = content
	}

	layers := []string{base}

	// Step 2: Append extension layers.
	globalExtendPath := filepath.Join(globalDir, agent+".prompt.md")
	if content, err := readIfExists(globalExtendPath); err == nil && content != "" {
		layers = append(layers, content)
	}

	localExtendPath := filepath.Join(localDir, agent+".prompt.md")
	if content, err := readIfExists(localExtendPath); err == nil && content != "" {
		layers = append(layers, content)
	}

	// Step 3: Concatenate with separator.
	return strings.Join(layers, PromptSeparator), nil
}

// CreateActivationMessage produces the full activation message for an agent,
// wrapping the resolved prompt with working directory and installation path
// context.
func (pr *PromptResolver) CreateActivationMessage(agent string) (string, error) {
	promptContent, err := pr.ResolvePrompt(agent)
	if err != nil {
		return "", fmt.Errorf("error loading prompt for %s: %w", agent, err)
	}

	displayName := strings.ToUpper(agent[:1]) + agent[1:]

	msg := fmt.Sprintf(
		"You are the %s Agent - read the prompt and follow the instructions exactly.\n\nWorking directory: %s\nSpekk installation: %s\n\nHere is your prompt:\n\n%s",
		displayName,
		pr.WorkDir,
		pr.InstallDir,
		promptContent,
	)

	return msg, nil
}

// readIfExists reads the file at path and returns its content. If the file
// does not exist it returns ("", nil). Any other I/O error is returned.
func readIfExists(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}
